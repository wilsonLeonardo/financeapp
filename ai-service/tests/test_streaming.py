import json

from langchain_core.messages import AIMessage, AIMessageChunk, ToolMessage

from app.streaming import stream_graph


class FakeGraph:
    def __init__(self, events: list, fail: bool = False):
        self.events = events
        self.fail = fail

    async def astream(self, inputs, config, stream_mode):
        for event in self.events:
            yield event
        if self.fail:
            raise RuntimeError("ollama down")


def parse(chunks: list[str]) -> list:
    out = []
    for chunk in chunks:
        assert chunk.startswith("data: ") and chunk.endswith("\n\n")
        data = chunk.removeprefix("data: ").strip()
        out.append(data if data == "[DONE]" else json.loads(data))
    return out


async def collect(graph: FakeGraph) -> list:
    return parse([c async for c in stream_graph(graph, {}, {})])


def token(text: str, node: str) -> tuple:
    return ("messages", (AIMessageChunk(content=text), {"langgraph_node": node}))


async def test_tool_round_trip_then_streamed_answer() -> None:
    call = {"id": "call-1", "name": "spending_by_category", "args": {"month": "2026-09"}}
    events = [
        token('{"route": "analyst"}', "router"),
        ("updates", {"router": {"route": "analyst"}}),
        ("updates", {"analyst": {"messages": [AIMessage(content="", tool_calls=[call])]}}),
        (
            "updates",
            {"analyst_tools": {"messages": [ToolMessage('{"total_spent": 10.5}', tool_call_id="call-1")]}},
        ),
        token("Você gastou ", "analyst"),
        token("R$ 10,50.", "analyst"),
        ("updates", {"analyst": {"messages": [AIMessage(content="Você gastou R$ 10,50.")]}}),
    ]

    chunks = await collect(FakeGraph(events))

    types = [c if c == "[DONE]" else c["type"] for c in chunks]
    assert types == [
        "start",
        "start-step", "tool-input-available", "tool-output-available", "finish-step",
        "start-step", "text-start", "text-delta", "text-delta", "text-end", "finish-step",
        "finish", "[DONE]",
    ]  # fmt: skip
    tool_input, tool_output = chunks[2], chunks[3]
    assert tool_input["toolName"] == "spending_by_category" and tool_input["input"] == {"month": "2026-09"}
    assert tool_output == {
        "type": "tool-output-available",
        "toolCallId": "call-1",
        "output": {"total_spent": 10.5},
        "dynamic": True,
    }
    deltas = "".join(c["delta"] for c in chunks if c != "[DONE]" and c["type"] == "text-delta")
    assert deltas == "Você gastou R$ 10,50."  # the router's JSON never reaches the user


async def test_failures_end_with_an_error_chunk_and_a_clean_finish() -> None:
    chunks = await collect(FakeGraph([token("Olá", "general")], fail=True))

    types = [c if c == "[DONE]" else c["type"] for c in chunks]
    assert types == [
        "start",
        "start-step",
        "text-start",
        "text-delta",
        "text-end",
        "finish-step",
        "error",
        "finish",
        "[DONE]",
    ]


async def test_narration_before_a_tool_call_is_held_back() -> None:
    call = {"id": "call-1", "name": "search_transactions", "args": {"category": "Casa"}}
    events = [
        token("Vou usar a função search_transactions.", "analyst"),
        ("updates", {"analyst": {"messages": [AIMessage(content="Vou usar...", tool_calls=[call])]}}),
        ("updates", {"analyst_tools": {"messages": [ToolMessage('{"count": 2}', tool_call_id="call-1")]}}),
        token("Foram 2 transações.", "analyst"),
        ("updates", {"analyst": {"messages": [AIMessage(content="Foram 2 transações.")]}}),
    ]

    chunks = await collect(FakeGraph(events))

    deltas = "".join(c["delta"] for c in chunks if c != "[DONE]" and c["type"] == "text-delta")
    assert deltas == "Foram 2 transações."


async def test_an_answer_that_never_used_a_tool_is_still_sent_whole() -> None:
    events = [
        token("Não ", "analyst"),
        token("encontrei dados.", "analyst"),
        ("updates", {"analyst": {"messages": [AIMessage(content="Não encontrei dados.")]}}),
    ]

    chunks = await collect(FakeGraph(events))

    deltas = [c["delta"] for c in chunks if c != "[DONE]" and c["type"] == "text-delta"]
    assert deltas == ["Não encontrei dados."]
