"""Retrieval-augmented categorization of bank transactions.

For each uncategorized transaction:

1. Retrieve the user's most similar already-categorized transactions from the vector index.
2. If the closest ones are near-identical and agree, take their category without calling the LLM.
   Recurring merchants are the bulk of a statement, so this path is fast, free and deterministic.
3. Otherwise ask the LLM to pick one of the user's categories, with those neighbours as examples.
   A JSON schema restricts the answer to a real category name or "unknown", so a small local
   model cannot invent categories.

The same class also runs the eval baselines: zero-shot (LLM, no retrieval) and kNN (no LLM).
"""

import asyncio
import logging
from collections import defaultdict
from collections.abc import Mapping, Sequence
from dataclasses import dataclass, field
from enum import StrEnum
from typing import Literal

from langchain_core.language_models import BaseChatModel
from langchain_core.messages import HumanMessage, SystemMessage

from app.finance_api import Category
from app.rag.normalize import index_text
from app.rag.store import Neighbor, TransactionIndex

log = logging.getLogger(__name__)

UNKNOWN = "unknown"

SYSTEM_PROMPT = (
    "You categorize bank transactions for a personal finance app. "
    'Pick exactly one category from the user\'s list, or "unknown" when none fits. '
    "Similar past transactions from the same user, when given, are the strongest evidence."
)


class Strategy(StrEnum):
    RAG = "rag"
    ZERO_SHOT = "zero_shot"
    KNN = "knn"


@dataclass(frozen=True)
class Transaction:
    id: str
    description: str
    txn_type: str
    amount: str


@dataclass
class Suggestion:
    expense_id: str
    category_id: str | None
    method: Literal["knn", "llm", "none"]
    confidence: float | None  # kNN similarity or vote share; None for LLM choices
    neighbors: list[Neighbor] = field(default_factory=list)


class Categorizer:
    def __init__(
        self,
        llm: BaseChatModel,
        index: TransactionIndex,
        *,
        k: int = 5,
        auto_threshold: float = 0.90,
        strategy: Strategy = Strategy.RAG,
    ):
        self.llm = llm
        self.index = index
        self.k = k
        self.auto_threshold = auto_threshold
        self.strategy = strategy

    async def categorize(
        self,
        user_id: str,
        txn: Transaction,
        categories: Sequence[Category],
        exemplars: Mapping[str, Sequence[str]] | None = None,
    ) -> Suggestion:
        """``exemplars`` maps category IDs to typical merchants from the user's history (RAG only)."""
        known = {c.id for c in categories}
        neighbors: list[Neighbor] = []
        if self.strategy != Strategy.ZERO_SHOT:
            query = index_text(txn.description)
            found = await asyncio.to_thread(self.index.search, user_id, txn.txn_type, query, self.k)
            # A category deleted after indexing would point nowhere.
            neighbors = [n for n in found if n.category_id in known]

        if self.strategy == Strategy.KNN:
            return self._vote(txn, neighbors)
        if self.strategy == Strategy.RAG and (agreed := self._agreed(neighbors)):
            return Suggestion(txn.id, agreed, "knn", neighbors[0].similarity, neighbors)
        if not categories:
            return Suggestion(txn.id, None, "none", None, neighbors)

        examples = exemplars if self.strategy == Strategy.RAG else None
        category_id = await self._ask_llm(txn, categories, neighbors, examples or {})
        return Suggestion(txn.id, category_id, "llm" if category_id else "none", None, neighbors)

    async def categorize_many(
        self,
        user_id: str,
        txns: Sequence[Transaction],
        categories: Sequence[Category],
        exemplars: Mapping[str, Sequence[str]] | None = None,
    ) -> list[Suggestion]:
        # Sequential on purpose: a CPU-bound local model gains nothing from concurrent requests.
        return [await self.categorize(user_id, t, categories, exemplars) for t in txns]

    def _agreed(self, neighbors: Sequence[Neighbor]) -> str | None:
        """The top neighbour's category when it is near-identical and most of the top three agree."""
        if not neighbors or neighbors[0].similarity < self.auto_threshold:
            return None
        top = neighbors[:3]
        votes = sum(1 for n in top if n.category_id == top[0].category_id)
        return top[0].category_id if votes * 2 > len(top) else None

    def _vote(self, txn: Transaction, neighbors: Sequence[Neighbor]) -> Suggestion:
        """kNN baseline: similarity-weighted majority over all neighbours."""
        if not neighbors:
            return Suggestion(txn.id, None, "none", None, list(neighbors))
        weights: dict[str, float] = defaultdict(float)
        for n in neighbors:
            weights[n.category_id] += n.similarity
        best = max(weights, key=weights.__getitem__)
        return Suggestion(txn.id, best, "knn", weights[best] / sum(weights.values()), list(neighbors))

    async def _ask_llm(
        self,
        txn: Transaction,
        categories: Sequence[Category],
        neighbors: Sequence[Neighbor],
        exemplars: Mapping[str, Sequence[str]],
    ) -> str | None:
        names = [c.name for c in categories]
        schema = {
            "title": "CategoryChoice",
            "description": "The category that best fits the transaction.",
            "type": "object",
            "properties": {"category": {"type": "string", "enum": [*names, UNKNOWN]}},
            "required": ["category"],
        }
        name_of = {c.id: c.name for c in categories}

        lines = [
            f"- {c.name} (e.g. {', '.join(exemplars[c.id])})" if exemplars.get(c.id) else f"- {c.name}"
            for c in categories
        ]
        prompt = "Categories:\n" + "\n".join(lines) + "\n\n"
        if neighbors:
            examples = "\n".join(f'- "{n.text}" -> {name_of[n.category_id]}' for n in neighbors)
            prompt += f"Similar past transactions from this user:\n{examples}\n\n"
        prompt += f'Transaction: "{txn.description}" ({txn.txn_type}, {txn.amount})\nWhich category?'

        try:
            structured = self.llm.with_structured_output(schema, method="json_schema")
            result = await structured.ainvoke([SystemMessage(SYSTEM_PROMPT), HumanMessage(prompt)])
        except Exception:
            log.warning("llm categorization failed", exc_info=True)
            return None

        choice = str((result or {}).get("category", "")).casefold()
        return {c.name.casefold(): c.id for c in categories}.get(choice)
