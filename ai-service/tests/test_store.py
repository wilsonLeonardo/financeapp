from app.rag.store import IndexedTransaction, MemoryTransactionIndex


def item(expense_id: str, text: str, category_id: str, txn_type: str = "expense") -> IndexedTransaction:
    return IndexedTransaction(expense_id, text, txn_type, category_id)


def test_search_never_returns_another_users_transactions(embeddings) -> None:
    index = MemoryTransactionIndex(embeddings)
    index.upsert("alice", [item("a1", "expense ifood restaurante", "food-a")])
    index.upsert("bob", [item("b1", "expense ifood restaurante", "food-b")])

    found = index.search("alice", "expense", "expense ifood restaurante", k=5)

    assert [n.expense_id for n in found] == ["a1"]


def test_search_keeps_expenses_and_income_apart(embeddings) -> None:
    index = MemoryTransactionIndex(embeddings)
    index.upsert(
        "alice",
        [
            item("e1", "expense mercado livre", "shopping"),
            item("i1", "income mercado livre", "refunds", "income"),
        ],
    )

    found = index.search("alice", "income", "income mercado livre", k=5)

    assert [n.expense_id for n in found] == ["i1"]


def test_identical_text_scores_near_one(embeddings) -> None:
    index = MemoryTransactionIndex(embeddings)
    index.upsert("alice", [item("a1", "expense uber trip", "transport")])

    assert index.search("alice", "expense", "expense uber trip", k=1)[0].similarity > 0.99


def test_upsert_replaces_a_recategorized_transaction(embeddings) -> None:
    index = MemoryTransactionIndex(embeddings)
    index.upsert("alice", [item("a1", "expense amazon", "shopping")])
    index.upsert("alice", [item("a1", "expense amazon", "books")])

    assert index.indexed(["a1"])["a1"]["category_id"] == "books"
    assert len(index.search("alice", "expense", "expense amazon", k=5)) == 1
