from app.finance_api import Expense
from app.rag.store import MemoryTransactionIndex
from app.rag.sync import category_exemplars, sync_index


def expense(eid: str, description: str, category_id: str) -> Expense:
    return Expense(
        id=eid, description=description, amount="10", type="expense", date="2026-09-01T00:00:00Z",
        category_id=category_id,
    )  # fmt: skip


async def test_sync_indexes_only_new_or_changed_transactions(embeddings) -> None:
    index = MemoryTransactionIndex(embeddings)
    history = [expense("e1", "UBER TRIP", "transport"), expense("e2", "PADARIA", "food")]

    assert await sync_index(index, "u1", history) == 2
    assert await sync_index(index, "u1", history) == 0

    recategorized = [expense("e1", "UBER TRIP", "work"), expense("e2", "PADARIA DOCE PAO", "food")]
    assert await sync_index(index, "u1", recategorized) == 2
    meta = index.indexed(["e1", "e2"])
    assert (meta["e1"]["category_id"], meta["e1"]["text"]) == ("work", "uber trip")
    assert (meta["e2"]["category_id"], meta["e2"]["text"]) == ("food", "padaria doce pao")


async def test_sync_drops_transactions_that_left_the_history(embeddings) -> None:
    index = MemoryTransactionIndex(embeddings)
    await sync_index(index, "u1", [expense("e1", "UBER TRIP", "transport"), expense("e2", "PADARIA", "food")])
    await sync_index(index, "u2", [expense("e9", "PADARIA", "food")])

    # e2 was deleted, or its category removed: it is no longer part of u1's history.
    assert await sync_index(index, "u1", [expense("e1", "UBER TRIP", "transport")]) == 0

    assert index.indexed_ids("u1") == {"e1"}
    assert index.indexed_ids("u2") == {"e9"}


def test_exemplars_are_each_categorys_most_frequent_merchants() -> None:
    history = [
        expense("1", "UBER *TRIP 01/09", "transport"),
        expense("2", "UBER *TRIP 4321", "transport"),
        expense("3", "POSTO SHELL", "transport"),
        expense("4", "PADARIA", "food"),
    ]

    exemplars = category_exemplars(history, per_category=1)

    assert exemplars == {"transport": ["uber trip"], "food": ["padaria"]}


async def test_sync_reembeds_rows_from_another_embedding_model(embeddings) -> None:
    history = [expense("e1", "UBER TRIP", "transport")]
    old = MemoryTransactionIndex(embeddings, model="old-model")
    await sync_index(old, "u1", history)

    new = MemoryTransactionIndex(embeddings, model="new-model")
    new.store = old.store  # same table, as with one pgvector collection

    assert await sync_index(new, "u1", history) == 1
    assert new.indexed(["e1"])["e1"]["model"] == "new-model"
