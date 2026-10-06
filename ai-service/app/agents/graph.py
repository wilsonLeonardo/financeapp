"""Multi-agent graph: a router hands each message to one specialist.

    START -> router -> analyst     <-> analyst_tools      -> END
                    -> categorizer <-> categorizer_tools  -> END
                    -> general                            -> END

The router is a single structured-output call, not a chatty agent, so it adds one short request
instead of a conversation. Each specialist only sees the tools it needs: a 3B local model picks the
right tool far more reliably from two or four options than from all of them.

Rewriting follow-ups into standalone questions first was tried and dropped: a 3B model drifts when
it rewrites, and the tool-choice eval fell from 81% to 76%.
"""

from collections.abc import Sequence
from datetime import date

from langchain_core.language_models import BaseChatModel
from langchain_core.messages import BaseMessage, HumanMessage, SystemMessage, ToolMessage
from langchain_core.tools import BaseTool
from langgraph.graph import END, START, MessagesState, StateGraph
from langgraph.graph.state import CompiledStateGraph
from langgraph.prebuilt import ToolNode, tools_condition

from app.agents import prompts

ROUTES = ("analyst", "categorizer", "general")
TEXT_NODES = frozenset(ROUTES)
GROUNDED_NODES = frozenset({"analyst", "categorizer"})  # specialists that must use a tool every turn
TOOL_NODES = frozenset({"analyst_tools", "categorizer_tools"})

GROUNDING_NUDGE = (
    "You have not called a tool for this question yet. Call one now to get the data: earlier replies in "
    "this conversation are summaries, not data."
)

_ROUTE_SCHEMA = {
    "title": "Route",
    "description": "Which assistant should answer.",
    "type": "object",
    "properties": {"route": {"type": "string", "enum": list(ROUTES)}},
    "required": ["route"],
}


class AgentState(MessagesState):
    route: str


def latest_question(messages: Sequence[BaseMessage]) -> str:
    return next((str(m.text) for m in reversed(messages) if isinstance(m, HumanMessage)), "")


async def choose_route(llm: BaseChatModel, question: str) -> str:
    """The specialist for a question; falls back to the analyst."""
    try:
        router = llm.with_structured_output(_ROUTE_SCHEMA, method="json_schema")
        result = await router.ainvoke([SystemMessage(prompts.ROUTER), HumanMessage(question)])
        route = (result or {}).get("route")
    except Exception:
        route = None
    return route if route in ROUTES else "analyst"


def used_tool_this_turn(messages: Sequence[BaseMessage]) -> bool:
    """Whether a tool already answered since the user's latest message."""
    for message in reversed(messages):
        if isinstance(message, ToolMessage):
            return True
        if isinstance(message, HumanMessage):
            return False
    return False


async def decide(
    model: BaseChatModel, system: str, messages: Sequence[BaseMessage], *, grounded: bool
) -> BaseMessage:
    """One specialist step.

    A grounded specialist must back every turn with a tool call. Asked a follow-up, a small model
    tends to answer from its own previous reply instead, and invents detail it never fetched; when
    its first reply in a turn calls no tool, it is asked once more, explicitly.
    """
    reply = await model.ainvoke([SystemMessage(system), *messages])
    if grounded and not getattr(reply, "tool_calls", None) and not used_tool_this_turn(messages):
        reply = await model.ainvoke([SystemMessage(f"{system}\n\n{GROUNDING_NUDGE}"), *messages])
    return reply


def build_graph(
    llm: BaseChatModel,
    analyst_tools: Sequence[BaseTool],
    categorizer_tools: Sequence[BaseTool],
    today: date,
    language: str = "English",
) -> CompiledStateGraph:
    def specialist(model: BaseChatModel, system: str, *, grounded: bool):
        async def node(state: AgentState) -> dict:
            return {"messages": [await decide(model, system, state["messages"], grounded=grounded)]}

        return node

    async def router(state: AgentState) -> dict:
        return {"route": await choose_route(llm, latest_question(state["messages"]))}

    graph = StateGraph(AgentState)
    graph.add_node("router", router)
    graph.add_node(
        "analyst", specialist(llm.bind_tools(analyst_tools), prompts.analyst(today, language), grounded=True)
    )
    graph.add_node("analyst_tools", ToolNode(analyst_tools))
    graph.add_node(
        "categorizer",
        specialist(llm.bind_tools(categorizer_tools), prompts.categorizer(today, language), grounded=True),
    )
    graph.add_node("categorizer_tools", ToolNode(categorizer_tools))
    graph.add_node("general", specialist(llm, prompts.general(today, language), grounded=False))

    graph.add_edge(START, "router")
    graph.add_conditional_edges("router", lambda s: s["route"], {r: r for r in ROUTES})
    for name in ("analyst", "categorizer"):
        graph.add_conditional_edges(name, tools_condition, {"tools": f"{name}_tools", END: END})
        graph.add_edge(f"{name}_tools", name)
    graph.add_edge("general", END)
    return graph.compile()
