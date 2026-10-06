import pytest
from fastapi import HTTPException
from langchain_core.messages import AIMessage, HumanMessage

from app.ratelimit import enforce_rate_limit
from app.schemas import MAX_HISTORY, UIMessage, to_langchain
from tests.conftest import FakeRedis


async def test_rate_limit_blocks_after_the_limit() -> None:
    redis = FakeRedis()
    for _ in range(2):
        await enforce_rate_limit(redis, "u1", limit=2)
    with pytest.raises(HTTPException) as exc:
        await enforce_rate_limit(redis, "u1", limit=2)
    assert exc.value.status_code == 429


async def test_rate_limit_is_per_user_and_fails_open() -> None:
    redis = FakeRedis()
    await enforce_rate_limit(redis, "u1", limit=1)
    await enforce_rate_limit(redis, "u2", limit=1)
    await enforce_rate_limit(FakeRedis(fail=True), "u1", limit=0)


def ui(role: str, *texts: str, other_parts: int = 0) -> UIMessage:
    parts = [{"type": "text", "text": t} for t in texts] + [{"type": "step-start"}] * other_parts
    return UIMessage(role=role, parts=parts)


def test_to_langchain_keeps_text_and_recent_history() -> None:
    history = [ui("user", f"q{i}") for i in range(MAX_HISTORY + 3)]
    history += [ui("assistant", "Olá, ", "tudo bem?", other_parts=2), ui("assistant", other_parts=1)]

    converted = to_langchain(history)

    assert len(converted) == MAX_HISTORY - 1  # the empty assistant message is dropped
    assert isinstance(converted[0], HumanMessage)
    assert converted[-1] == AIMessage("Olá, tudo bem?")
