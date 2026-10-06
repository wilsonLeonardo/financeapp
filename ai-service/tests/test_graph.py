from langchain_core.messages import AIMessage, HumanMessage, ToolMessage

from app.agents.graph import GROUNDING_NUDGE, decide, used_tool_this_turn


class ScriptedModel:
    """Returns scripted replies and records the system prompt of every call."""

    def __init__(self, replies: list[AIMessage]):
        self.replies = list(replies)
        self.systems: list[str] = []

    async def ainvoke(self, messages):
        self.systems.append(messages[0].content)
        return self.replies.pop(0)


def tool_call(name: str) -> AIMessage:
    return AIMessage(content="", tool_calls=[{"id": "c1", "name": name, "args": {}}])


async def test_an_ungrounded_first_reply_is_retried_with_a_nudge() -> None:
    model = ScriptedModel([AIMessage("Casa had $1,753.51 of Mercado..."), tool_call("search_transactions")])

    reply = await decide(model, "SYSTEM", [HumanMessage("details of Casa?")], grounded=True)

    assert reply.tool_calls[0]["name"] == "search_transactions"
    assert GROUNDING_NUDGE in model.systems[1] and GROUNDING_NUDGE not in model.systems[0]


async def test_no_retry_once_a_tool_answered_or_for_ungrounded_specialists() -> None:
    after_tool = [
        HumanMessage("details?"),
        tool_call("search_transactions"),
        ToolMessage("{}", tool_call_id="c1"),
    ]
    model = ScriptedModel([AIMessage("Here are the details.")])
    assert (await decide(model, "S", after_tool, grounded=True)).content == "Here are the details."

    chatty = ScriptedModel([AIMessage("Hi!")])
    assert (await decide(chatty, "S", [HumanMessage("oi")], grounded=False)).content == "Hi!"
    assert len(model.systems) == len(chatty.systems) == 1


def test_used_tool_this_turn_only_looks_at_the_latest_question() -> None:
    earlier_turn = [HumanMessage("q1"), tool_call("x"), ToolMessage("{}", tool_call_id="c1"), AIMessage("a1")]
    assert used_tool_this_turn(earlier_turn) is True
    assert used_tool_this_turn([*earlier_turn, HumanMessage("q2")]) is False
