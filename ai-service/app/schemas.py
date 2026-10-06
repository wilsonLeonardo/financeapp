from typing import Literal

from langchain_core.messages import AIMessage, BaseMessage, HumanMessage
from pydantic import BaseModel, ConfigDict, Field

MAX_HISTORY = 12  # messages sent to the model; a small local context window fills up fast


class UIPart(BaseModel):
    model_config = ConfigDict(extra="allow")
    type: str
    text: str | None = None


class UIMessage(BaseModel):
    """A message as useChat sends it. Only text parts are forwarded to the model."""

    model_config = ConfigDict(extra="allow")
    id: str | None = None
    role: Literal["user", "assistant", "system"]
    parts: list[UIPart] = Field(default_factory=list)

    @property
    def text(self) -> str:
        return "".join(p.text or "" for p in self.parts if p.type == "text").strip()


class ChatRequest(BaseModel):
    model_config = ConfigDict(extra="allow")
    messages: list[UIMessage] = Field(min_length=1)


def to_langchain(messages: list[UIMessage]) -> list[BaseMessage]:
    converted: list[BaseMessage] = []
    for m in messages[-MAX_HISTORY:]:
        if not m.text:
            continue
        if m.role == "user":
            converted.append(HumanMessage(m.text))
        elif m.role == "assistant":
            converted.append(AIMessage(m.text))
    return converted


class SuggestRequest(BaseModel):
    limit: int = Field(50, ge=1, le=200)


class SuggestionOut(BaseModel):
    expense_id: str
    description: str
    amount: str
    type: str
    date: str
    category_id: str | None
    category_name: str | None
    method: Literal["knn", "llm", "none"]
    confidence: float | None


class SuggestResponse(BaseModel):
    suggestions: list[SuggestionOut]
    indexed: int


class ApplyItem(BaseModel):
    expense_id: str
    category_id: str


class ApplyRequest(BaseModel):
    items: list[ApplyItem] = Field(min_length=1, max_length=200)


class ApplyResponse(BaseModel):
    applied: int
    failed: list[str]
