"""Bridges the LangGraph run to the Vercel AI SDK UI message stream protocol (v1).

The React front end uses ``useChat`` from ``@ai-sdk/react``, which expects Server-Sent Events of
JSON chunks: ``start``, then per LLM call a step (``start-step`` ... ``finish-step``) holding text
parts (``text-start``/``text-delta``/``text-end``) and tool parts (``tool-input-available`` then
``tool-output-available``), and finally ``finish`` and ``[DONE]``.

Tokens come from LangGraph's "messages" stream mode; complete tool calls and tool results come from
"updates", once each node finishes.

Text from a grounded specialist is only streamed once a tool has answered in this run. Before that,
its words are either narration of the call it is about to make or an answer with no data behind it
(which the graph retries), so they are held back; an answer that never got grounded is still sent
whole when the specialist finishes.
"""

import json
import logging
import uuid
from collections.abc import AsyncIterator, Iterator
from typing import Any

from langchain_core.messages import AIMessageChunk, BaseMessage, ToolMessage
from langgraph.graph.state import CompiledStateGraph

from app.agents.graph import GROUNDED_NODES, TEXT_NODES, TOOL_NODES
from app.errors import describe_failure

log = logging.getLogger(__name__)

HEADERS = {
    "x-vercel-ai-ui-message-stream": "v1",
    "cache-control": "no-cache",
    "x-accel-buffering": "no",  # keeps nginx from buffering the stream
}


def sse(payload: dict[str, Any] | str) -> str:
    data = payload if isinstance(payload, str) else json.dumps(payload, ensure_ascii=False, default=str)
    return f"data: {data}\n\n"


class UIMessageStream:
    """Tracks open steps and text parts so every chunk sequence it emits is well formed."""

    def __init__(self) -> None:
        self._step_open = False
        self._text_id: str | None = None

    def start(self) -> Iterator[str]:
        yield sse({"type": "start", "messageId": f"msg-{uuid.uuid4().hex}"})

    def text(self, delta: str) -> Iterator[str]:
        yield from self._ensure_step()
        if self._text_id is None:
            self._text_id = f"text-{uuid.uuid4().hex}"
            yield sse({"type": "text-start", "id": self._text_id})
        yield sse({"type": "text-delta", "id": self._text_id, "delta": delta})

    def tool_input(self, call_id: str, name: str, args: Any) -> Iterator[str]:
        yield from self._ensure_step()
        yield from self._end_text()
        yield sse(
            {
                "type": "tool-input-available",
                "toolCallId": call_id,
                "toolName": name,
                "input": args,
                "dynamic": True,
            }
        )

    def tool_output(self, call_id: str, output: Any) -> Iterator[str]:
        yield sse({"type": "tool-output-available", "toolCallId": call_id, "output": output, "dynamic": True})

    def finish_step(self) -> Iterator[str]:
        yield from self._end_text()
        if self._step_open:
            self._step_open = False
            yield sse({"type": "finish-step"})

    def error(self, message: str) -> Iterator[str]:
        yield from self.finish_step()
        yield sse({"type": "error", "errorText": message})

    def finish(self) -> Iterator[str]:
        yield from self.finish_step()
        yield sse({"type": "finish"})
        yield sse("[DONE]")

    def _ensure_step(self) -> Iterator[str]:
        if not self._step_open:
            self._step_open = True
            yield sse({"type": "start-step"})

    def _end_text(self) -> Iterator[str]:
        if self._text_id is not None:
            yield sse({"type": "text-end", "id": self._text_id})
            self._text_id = None


def _tool_output(message: ToolMessage) -> Any:
    try:
        return json.loads(message.content) if isinstance(message.content, str) else message.content
    except ValueError:
        return message.content


def _node_messages(update: Any) -> list[BaseMessage]:
    return list((update or {}).get("messages", [])) if isinstance(update, dict) else []


async def stream_graph(graph: CompiledStateGraph, inputs: dict, config: dict) -> AsyncIterator[str]:
    stream = UIMessageStream()
    grounded = False  # a tool has answered in this run
    for chunk in stream.start():
        yield chunk
    try:
        async for mode, payload in graph.astream(inputs, config, stream_mode=["messages", "updates"]):
            if mode == "messages":
                message, meta = payload
                # The router's tokens are its JSON decision, not something to show.
                node = meta.get("langgraph_node")
                if node in GROUNDED_NODES and not grounded:
                    continue
                if node in TEXT_NODES and isinstance(message, AIMessageChunk):
                    if text := str(message.text):
                        for chunk in stream.text(text):
                            yield chunk
                continue

            for node, update in payload.items():
                messages = _node_messages(update)
                if node in TEXT_NODES and messages:
                    calls = getattr(messages[-1], "tool_calls", None) or []
                    for call in calls:
                        for chunk in stream.tool_input(call["id"], call["name"], call["args"]):
                            yield chunk
                    if not calls:  # the answer is complete; tools would continue the step
                        if node in GROUNDED_NODES and not grounded and (text := str(messages[-1].text)):
                            for chunk in stream.text(text):
                                yield chunk
                        for chunk in stream.finish_step():
                            yield chunk
                elif node in TOOL_NODES:
                    grounded = True
                    for message in messages:
                        if isinstance(message, ToolMessage):
                            for chunk in stream.tool_output(message.tool_call_id, _tool_output(message)):
                                yield chunk
                    for chunk in stream.finish_step():
                        yield chunk
    except Exception as exc:
        log.exception("chat run failed")
        for chunk in stream.error(describe_failure(exc)):
            yield chunk
    for chunk in stream.finish():
        yield chunk
