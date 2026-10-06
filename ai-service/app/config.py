from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Every value is read from the environment and falls back to a local-development default."""

    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    # Go API the tools call on the user's behalf.
    finance_api_url: str = "http://localhost:8080/api/v1"

    # Local models served by Ollama.
    ollama_base_url: str = "http://localhost:11434"
    chat_model: str = "qwen2.5:3b"
    embed_model: str = "nomic-embed-text"
    # nomic-embed-text is trained with task prefixes; "classification: " suits symmetric
    # transaction-to-transaction similarity. Leave empty for models without prefixes.
    embed_prefix: str = "classification: "
    num_ctx: int = 4096

    # pgvector store for the categorization index.
    database_url: str = "postgresql+psycopg://financeapp:financeapp123@localhost:5432/financeapp"
    vector_collection: str = "transactions"

    # Same Redis and JWT secret as the Go API: tokens are verified here, and logout is honoured
    # through the blacklist the Go API writes.
    redis_url: str = "redis://:redis123@localhost:6379/0"
    jwt_secret: str = "default-secret-change-in-production"
    rate_limit_per_minute: int = 20

    cors_origins: list[str] = ["http://localhost:3000", "http://localhost:5173"]
    # "Today" and "this month" in prompts and tools follow the user's clock, not the container's UTC.
    timezone: str = "America/Sao_Paulo"

    # Retrieval-augmented categorization.
    knn_k: int = 5
    knn_auto_threshold: float = 0.90


@lru_cache
def get_settings() -> Settings:
    return Settings()
