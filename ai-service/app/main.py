import asyncio
import logging
from contextlib import asynccontextmanager
from datetime import date, datetime
from typing import Annotated
from zoneinfo import ZoneInfo

import httpx
from fastapi import Depends, FastAPI, Header, HTTPException, Request
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse, StreamingResponse
from ollama import ResponseError
from redis.asyncio import Redis
from starlette.exceptions import HTTPException as StarletteHTTPException

from app.agents.graph import build_graph
from app.agents.prompts import reply_language
from app.agents.tools import analyst_tools, categorizer_tools
from app.auth import Principal, authenticate
from app.config import get_settings
from app.errors import describe_failure
from app.finance_api import UNCATEGORIZED, FinanceAPI, FinanceAPIError
from app.llm import chat_model, embeddings
from app.rag.categorizer import Categorizer, Transaction
from app.rag.normalize import index_text
from app.rag.store import IndexedTransaction, PgTransactionIndex, TransactionIndex
from app.rag.sync import categorized_history, category_exemplars, sync_index
from app.ratelimit import enforce_rate_limit
from app.schemas import (
    ApplyRequest,
    ApplyResponse,
    ChatRequest,
    SuggestionOut,
    SuggestRequest,
    SuggestResponse,
    to_langchain,
)
from app.streaming import HEADERS, stream_graph

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
log = logging.getLogger("ai-service")

RECURSION_LIMIT = 12  # bounds the specialist <-> tools loop of one chat turn


@asynccontextmanager
async def lifespan(app: FastAPI):
    settings = get_settings()
    app.state.http = httpx.AsyncClient(base_url=settings.finance_api_url, timeout=30.0)
    app.state.redis = Redis.from_url(settings.redis_url, decode_responses=True)
    app.state.llm = chat_model(settings)
    app.state.index = None
    try:
        app.state.index = await asyncio.to_thread(
            PgTransactionIndex,
            embeddings(settings),
            settings.database_url,
            settings.vector_collection,
            settings.embed_model,
        )
    except Exception:
        # The chat analyst still works without the index; only categorization needs it.
        log.exception("vector index unavailable, categorization is disabled")
    yield
    await app.state.http.aclose()
    await app.state.redis.aclose()


app = FastAPI(title="FinanceApp AI service", lifespan=lifespan)
app.add_middleware(
    CORSMiddleware,
    allow_origins=get_settings().cors_origins,
    allow_methods=["*"],
    allow_headers=["*"],
    expose_headers=["x-vercel-ai-ui-message-stream"],
)


def _error(status_code: int, message: str, headers: dict[str, str] | None = None) -> JSONResponse:
    # Same envelope as the Go API, so the front end reads every error the same way.
    return JSONResponse(status_code=status_code, content={"message": message}, headers=headers)


@app.exception_handler(StarletteHTTPException)
async def http_error(_: Request, exc: StarletteHTTPException) -> JSONResponse:
    return _error(exc.status_code, str(exc.detail), getattr(exc, "headers", None))


@app.exception_handler(RequestValidationError)
async def validation_error(_: Request, exc: RequestValidationError) -> JSONResponse:
    first = exc.errors()[0] if exc.errors() else {}
    field = ".".join(str(part) for part in first.get("loc", ()))
    return _error(422, f"invalid request: {field} {first.get('msg', '')}".strip())


@app.exception_handler(FinanceAPIError)
async def finance_api_error(_: Request, exc: FinanceAPIError) -> JSONResponse:
    return _error(exc.status_code if 400 <= exc.status_code < 500 else 502, exc.message)


@app.exception_handler(ResponseError)
@app.exception_handler(ConnectionError)
async def model_unavailable(_: Request, exc: Exception) -> JSONResponse:
    log.warning("local model unavailable: %s", exc)
    return _error(503, describe_failure(exc))


async def current_user(request: Request, authorization: Annotated[str | None, Header()] = None) -> Principal:
    return await authenticate(authorization, get_settings().jwt_secret, request.app.state.redis)


User = Annotated[Principal, Depends(current_user)]


def finance_api(request: Request, user: User) -> FinanceAPI:
    return FinanceAPI(request.app.state.http, user.token)


API = Annotated[FinanceAPI, Depends(finance_api)]


def vector_index(request: Request) -> TransactionIndex:
    if request.app.state.index is None:
        raise HTTPException(503, "vector index unavailable")
    return request.app.state.index


Index = Annotated[TransactionIndex, Depends(vector_index)]


def make_categorizer(request: Request, index: TransactionIndex) -> Categorizer:
    settings = get_settings()
    return Categorizer(
        request.app.state.llm, index, k=settings.knn_k, auto_threshold=settings.knn_auto_threshold
    )


def today() -> date:
    return datetime.now(ZoneInfo(get_settings().timezone)).date()


@app.get("/health")
async def health(request: Request) -> dict:
    settings = get_settings()
    try:
        async with httpx.AsyncClient(timeout=3.0) as client:
            tags = (await client.get(f"{settings.ollama_base_url}/api/tags")).json()
        models, ollama = {m["name"] for m in tags.get("models", [])}, "up"
    except (httpx.HTTPError, ValueError):
        models, ollama = set(), "down"

    def pulled(model: str) -> bool:
        return model in models or f"{model}:latest" in models

    return {
        "status": "ok",
        "ollama": ollama,
        "models": {
            settings.chat_model: pulled(settings.chat_model),
            settings.embed_model: pulled(settings.embed_model),
        },
        "vector_index": "up" if request.app.state.index else "down",
    }


@app.post("/chat")
async def chat(body: ChatRequest, request: Request, user: User, api: API) -> StreamingResponse:
    await enforce_rate_limit(request.app.state.redis, user.user_id, get_settings().rate_limit_per_minute)
    messages = to_langchain(body.messages)
    if not messages:
        raise HTTPException(400, "no text messages to answer")

    index = request.app.state.index
    categorizer = make_categorizer(request, index) if index else None
    day = today()
    graph = build_graph(
        request.app.state.llm,
        analyst_tools(api, day),
        categorizer_tools(api, categorizer, user.user_id),
        day,
        reply_language(body.messages[-1].text),
    )
    return StreamingResponse(
        stream_graph(graph, {"messages": messages}, {"recursion_limit": RECURSION_LIMIT}),
        media_type="text/event-stream",
        headers=HEADERS,
    )


@app.post("/categorize/suggest", response_model=SuggestResponse)
async def suggest(
    body: SuggestRequest, request: Request, user: User, api: API, index: Index
) -> SuggestResponse:
    await enforce_rate_limit(request.app.state.redis, user.user_id, get_settings().rate_limit_per_minute)
    history = await categorized_history(api)
    indexed = await sync_index(index, user.user_id, history)

    pending = await api.expenses(limit=body.limit, category_id=UNCATEGORIZED)

    categories = await api.categories()
    names = {c.id: c.name for c in categories}
    txns = [Transaction(e.id, e.description, e.type, str(e.amount)) for e in pending]
    suggestions = await make_categorizer(request, index).categorize_many(
        user.user_id, txns, categories, category_exemplars(history)
    )

    return SuggestResponse(
        indexed=indexed,
        suggestions=[
            SuggestionOut(
                expense_id=e.id,
                description=e.description,
                amount=str(e.amount),
                type=e.type,
                date=e.day,
                category_id=s.category_id,
                category_name=names.get(s.category_id) if s.category_id else None,
                method=s.method,
                confidence=round(s.confidence, 3) if s.confidence is not None else None,
            )
            for e, s in zip(pending, suggestions, strict=True)
        ],
    )


@app.post("/categorize/apply", response_model=ApplyResponse)
async def apply(body: ApplyRequest, user: User, api: API, index: Index) -> ApplyResponse:
    # The Go API does not check that a category belongs to the caller, so it is checked here.
    valid = {c.id for c in await api.categories()}
    applied: list[IndexedTransaction] = []
    failed: list[str] = []
    for item in body.items:
        if item.category_id not in valid:
            failed.append(item.expense_id)
            continue
        try:
            expense = await api.set_category(item.expense_id, item.category_id)
        except FinanceAPIError:
            failed.append(item.expense_id)
            continue
        applied.append(
            IndexedTransaction(expense.id, index_text(expense.description), expense.type, item.category_id)
        )

    # Confirmed choices feed the index right away, so the next suggestions already learn from them.
    await asyncio.to_thread(index.upsert, user.user_id, applied)
    return ApplyResponse(applied=len(applied), failed=failed)


@app.post("/index/sync")
async def sync(user: User, api: API, index: Index) -> dict:
    return {"indexed": await sync_index(index, user.user_id, await categorized_history(api))}
