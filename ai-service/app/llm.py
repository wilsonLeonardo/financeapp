from langchain_core.embeddings import Embeddings
from langchain_ollama import ChatOllama, OllamaEmbeddings

from app.config import Settings


def chat_model(settings: Settings) -> ChatOllama:
    # Temperature 0 keeps tool calls and categorizations reproducible, which the evals rely on.
    return ChatOllama(
        model=settings.chat_model,
        base_url=settings.ollama_base_url,
        temperature=0,
        seed=42,
        num_ctx=settings.num_ctx,
    )


class PrefixedEmbeddings(Embeddings):
    """Adds the same task prefix to documents and queries.

    Transaction-to-transaction similarity is symmetric, so both sides get the prefix; nomic-embed-text
    is trained to expect one.
    """

    def __init__(self, base: Embeddings, prefix: str):
        self.base = base
        self.prefix = prefix

    def embed_documents(self, texts: list[str]) -> list[list[float]]:
        return self.base.embed_documents([self.prefix + t for t in texts])

    def embed_query(self, text: str) -> list[float]:
        return self.base.embed_query(self.prefix + text)


def embeddings(settings: Settings) -> Embeddings:
    base = OllamaEmbeddings(model=settings.embed_model, base_url=settings.ollama_base_url)
    return PrefixedEmbeddings(base, settings.embed_prefix) if settings.embed_prefix else base
