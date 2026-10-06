import httpx
import pytest
import respx
from fastapi.testclient import TestClient

import app.main as main
from app.rag.store import MemoryTransactionIndex
from tests.conftest import API, FakeRedis, FakeStructuredLLM, make_token
from tests.test_streaming import FakeGraph, token

AUTH = {"Authorization": f"Bearer {make_token('u1')}"}
CATEGORIES = [{"id": "c-food", "name": "Alimentação"}, {"id": "c-transport", "name": "Transporte"}]


def row(eid: str, description: str, **extra) -> dict:
    return {"id": eid, "description": description, "amount": "25.00", "type": "expense",
            "date": "2026-09-10T00:00:00Z", "tags": "", **extra}  # fmt: skip


@pytest.fixture
def client(embeddings):
    with TestClient(main.app) as c:
        main.app.state.redis = FakeRedis()
        main.app.state.index = MemoryTransactionIndex(embeddings)
        main.app.state.llm = FakeStructuredLLM([])
        yield c


def test_health_reports_dependencies(client: TestClient) -> None:
    body = client.get("/health").json()
    assert body["status"] == "ok"
    assert body["ollama"] == "down"


def test_endpoints_require_a_token(client: TestClient) -> None:
    assert client.post("/chat", json={"messages": [{"role": "user", "parts": []}]}).status_code == 401
    assert client.post("/categorize/suggest", json={}).status_code == 401


def test_chat_streams_the_ui_message_protocol(client: TestClient, monkeypatch) -> None:
    monkeypatch.setattr(main, "build_graph", lambda *a, **k: FakeGraph([token("Oi!", "general")]))

    resp = client.post(
        "/chat",
        headers=AUTH,
        json={"messages": [{"role": "user", "parts": [{"type": "text", "text": "oi"}]}]},
    )

    assert resp.status_code == 200
    assert resp.headers["x-vercel-ai-ui-message-stream"] == "v1"
    assert '"delta": "Oi!"' in resp.text and resp.text.rstrip().endswith("data: [DONE]")


@respx.mock
def test_suggest_covers_uncategorized_transactions_up_to_the_limit(client: TestClient) -> None:
    def expenses(request: httpx.Request) -> httpx.Response:
        if request.url.params.get("category_id") == "none":
            data = [row("p1", "IFOOD *RESTAURANTE 4321"), row("p2", "UBER TRIP")]
        else:
            data = [
                row("h1", "IFOOD *RESTAURANTE 1111", category_id="c-food"),
                row("p1", "IFOOD *RESTAURANTE 4321"),
                row("p2", "UBER TRIP"),
            ]
        return httpx.Response(200, json={"data": data, "total": len(data), "page": 1, "page_size": 100})

    respx.get(f"{API}/expenses").mock(side_effect=expenses)
    respx.get(f"{API}/categories").mock(return_value=httpx.Response(200, json=CATEGORIES))

    # p2 would need the LLM, which the fake has no answers for: the limit keeps it out.
    body = client.post("/categorize/suggest", headers=AUTH, json={"limit": 1}).json()

    assert body["indexed"] == 1  # only the categorized history row is indexed
    (suggestion,) = body["suggestions"]
    assert (suggestion["expense_id"], suggestion["category_name"], suggestion["method"]) == (
        "p1",
        "Alimentação",
        "knn",
    )


@respx.mock
def test_apply_rejects_foreign_categories_and_indexes_the_rest(client: TestClient) -> None:
    respx.get(f"{API}/categories").mock(return_value=httpx.Response(200, json=CATEGORIES))
    respx.get(f"{API}/expenses/e1").mock(return_value=httpx.Response(200, json=row("e1", "POSTO SHELL")))
    respx.put(f"{API}/expenses/e1").mock(
        return_value=httpx.Response(200, json=row("e1", "POSTO SHELL", category_id="c-transport"))
    )

    body = client.post(
        "/categorize/apply",
        headers=AUTH,
        json={
            "items": [
                {"expense_id": "e1", "category_id": "c-transport"},
                {"expense_id": "e2", "category_id": "c-other"},
            ]
        },
    ).json()

    assert body == {"applied": 1, "failed": ["e2"]}
    assert main.app.state.index.indexed(["e1"])["e1"]["category_id"] == "c-transport"


def test_categorization_needs_the_vector_index(client: TestClient) -> None:
    main.app.state.index = None
    assert client.post("/index/sync", headers=AUTH).status_code == 503


def test_errors_use_the_go_api_envelope(client: TestClient) -> None:
    resp = client.post("/categorize/suggest", json={})
    assert resp.status_code == 401
    assert resp.json() == {"message": "authorization header required"}

    resp = client.post("/categorize/suggest", headers=AUTH, json={"limit": 0})
    assert resp.status_code == 422 and resp.json()["message"].startswith("invalid request: body.limit")


def test_missing_model_is_a_503_with_instructions(client: TestClient, monkeypatch) -> None:
    from ollama import ResponseError

    async def no_model(*_args, **_kwargs):
        raise ResponseError("model 'nomic-embed-text' not found", 404)

    monkeypatch.setattr(main, "categorized_history", no_model)

    resp = client.post("/categorize/suggest", headers=AUTH, json={})

    assert resp.status_code == 503
    assert "nomic-embed-text" in resp.json()["message"] and "make up" in resp.json()["message"]
