from ollama import ResponseError

from app.errors import describe_failure


def test_missing_model_says_how_to_get_it() -> None:
    message = describe_failure(ResponseError("model 'qwen2.5:3b' not found", 404))
    assert "qwen2.5:3b" in message and "make up" in message


def test_unreachable_ollama_and_anything_else() -> None:
    assert "Cannot reach" in describe_failure(ConnectionError("refused"))
    assert "ai-service logs" in describe_failure(RuntimeError("boom"))
