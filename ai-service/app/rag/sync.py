import asyncio
from collections import Counter, defaultdict
from collections.abc import Sequence

from app.finance_api import Expense, FinanceAPI
from app.rag.normalize import index_text, normalize_description
from app.rag.store import IndexedTransaction, TransactionIndex

_BATCH = 200  # embeddings per call; keeps a first sync of a long history from one huge request


async def categorized_history(api: FinanceAPI, *, limit: int = 5000) -> list[Expense]:
    """The user's transactions that already have a category: the categorizer's ground truth.

    Only the latest ``limit`` transactions are read, which also bounds the index (see sync_index).
    """
    return [e for e in await api.expenses(limit=limit) if e.category_id]


async def sync_index(index: TransactionIndex, user_id: str, history: Sequence[Expense]) -> int:
    """Makes the index mirror the user's categorized history.

    Indexes the transactions that are new, re-categorized or re-described since the last sync, and
    drops the ones that left the history (deleted, or no longer categorized) so they stop showing up
    as neighbours. Categories assigned anywhere in the app reach the index this way, so it needs no
    hooks in the Go API. Comparing the stored text and model also re-embeds everything after a
    normalization or embedding-model change.
    Returns how many transactions were (re)indexed.
    """
    current = {
        e.id: IndexedTransaction(e.id, index_text(e.description), e.type, e.category_id or "")
        for e in history
    }
    known = await asyncio.to_thread(index.indexed, list(current))
    stale = [
        item
        for expense_id, item in current.items()
        if (meta := known.get(expense_id)) is None
        or meta.get("category_id") != item.category_id
        or meta.get("text") != item.text
        or meta.get("model") != index.model
    ]
    for start in range(0, len(stale), _BATCH):
        await asyncio.to_thread(index.upsert, user_id, stale[start : start + _BATCH])

    gone = (await asyncio.to_thread(index.indexed_ids, user_id)).difference(current)
    if gone:
        await asyncio.to_thread(index.delete, sorted(gone))
    return len(stale)


def category_exemplars(history: Sequence[Expense], per_category: int = 6) -> dict[str, list[str]]:
    """The most frequent merchants per category in the user's history.

    Category names alone are ambiguous to a small model ("Mercado" or "Compras"?); a few examples of
    what this user files under each one let it place merchants it has never seen.
    """
    counts: dict[str, Counter[str]] = defaultdict(Counter)
    for e in history:
        counts[e.category_id or ""][normalize_description(e.description)] += 1
    return {category: [text for text, _ in c.most_common(per_category)] for category, c in counts.items()}
