"""Messages for failures of the local model stack that tell the user what to do about them."""

import httpx
from ollama import ResponseError


def describe_failure(exc: BaseException) -> str:
    """A message the user can act on; the full traceback stays in the logs."""
    if isinstance(exc, ResponseError) and exc.status_code == 404:
        return (
            f"{exc.error}: the model is not in Ollama yet. Restart the stack (make up) to download it, "
            "or run make ai-models."
        )
    if isinstance(exc, ConnectionError | httpx.ConnectError):
        return "Cannot reach the local model server (Ollama). Is it running?"
    return "The assistant failed to answer. Check the ai-service logs."
