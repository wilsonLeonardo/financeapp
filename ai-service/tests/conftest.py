"""Test doubles and environment. Nothing here needs Ollama, Postgres or Redis running."""

import hashlib
import math
import os
import time

# Point every external dependency at closed ports before the app reads its settings, so the suite is
# hermetic: Redis and Postgres fail fast (and the app degrades), the Go API is mocked with respx.
os.environ.update(
    {
        "FINANCE_API_URL": "http://finance.test/api/v1",
        "OLLAMA_BASE_URL": "http://127.0.0.1:1",
        "DATABASE_URL": "postgresql+psycopg://test:test@127.0.0.1:1/test",
        "REDIS_URL": "redis://127.0.0.1:1/0",
        "JWT_SECRET": "test-secret",
    }
)

import jwt  # noqa: E402
import pytest  # noqa: E402
from langchain_core.embeddings import Embeddings  # noqa: E402

API = "http://finance.test/api/v1"
SECRET = "test-secret"


def make_token(user_id: str = "user-1", secret: str = SECRET, expires_in: int = 3600) -> str:
    now = int(time.time())
    return jwt.encode({"user_id": user_id, "iat": now, "exp": now + expires_in}, secret, algorithm="HS256")


class HashEmbeddings(Embeddings):
    """Deterministic bag-of-words vectors: same words, same vector; shared words, high similarity."""

    dim = 64

    def _embed(self, text: str) -> list[float]:
        vec = [0.0] * self.dim
        for token in text.split():
            vec[int(hashlib.md5(token.encode()).hexdigest(), 16) % self.dim] += 1.0
        norm = math.sqrt(sum(x * x for x in vec)) or 1.0
        return [x / norm for x in vec]

    def embed_documents(self, texts: list[str]) -> list[list[float]]:
        return [self._embed(t) for t in texts]

    def embed_query(self, text: str) -> list[float]:
        return self._embed(text)


class FakeStructuredLLM:
    """Stands in for ChatOllama where only structured output is used; records every prompt."""

    def __init__(self, answers: list[dict | Exception]):
        self.answers = list(answers)
        self.calls: list[list] = []
        self.schemas: list[dict] = []

    def with_structured_output(self, schema: dict, method: str = "json_schema"):
        self.schemas.append(schema)
        outer = self

        class _Runnable:
            async def ainvoke(self, messages):
                outer.calls.append(messages)
                answer = outer.answers.pop(0)
                if isinstance(answer, Exception):
                    raise answer
                return answer

        return _Runnable()


class FakeRedis:
    def __init__(self, blacklist: set[str] | None = None, fail: bool = False):
        self.blacklist = blacklist or set()
        self.fail = fail
        self.counters: dict[str, int] = {}

    def _check(self) -> None:
        if self.fail:
            from redis.exceptions import ConnectionError

            raise ConnectionError("redis down")

    async def exists(self, key: str) -> int:
        self._check()
        return int(key.removeprefix("blacklist:") in self.blacklist)

    async def incr(self, key: str) -> int:
        self._check()
        self.counters[key] = self.counters.get(key, 0) + 1
        return self.counters[key]

    async def expire(self, key: str, seconds: int) -> bool:
        return True

    async def aclose(self) -> None:
        pass


@pytest.fixture
def embeddings() -> HashEmbeddings:
    return HashEmbeddings()
