"""Runs the pgvector index against a real database; skipped unless PGVECTOR_TEST_URL is set.

The in-memory index shares the interface but not PGVector's storage rules (IDs are unique across
collections, metadata filters compile to SQL), so this is the test that covers them. CI runs it
against a pgvector service container.
"""

import os
import uuid

import pytest

from app.rag.store import IndexedTransaction, PgTransactionIndex

URL = os.environ.get("PGVECTOR_TEST_URL")
pytestmark = pytest.mark.skipif(not URL, reason="set PGVECTOR_TEST_URL to run against a real pgvector")


def item(expense_id: str, text: str, category_id: str) -> IndexedTransaction:
    return IndexedTransaction(expense_id, text, "expense", category_id)


@pytest.fixture
def index(embeddings) -> PgTransactionIndex:
    return PgTransactionIndex(embeddings, URL, f"test-{uuid.uuid4().hex}", model="hash")


def test_search_is_scoped_to_the_user_in_sql(index: PgTransactionIndex) -> None:
    mine, theirs = f"a-{uuid.uuid4().hex}", f"b-{uuid.uuid4().hex}"
    index.upsert("alice", [item(mine, "ifood restaurante", "food")])
    index.upsert("bob", [item(theirs, "ifood restaurante", "food")])

    found = index.search("alice", "expense", "ifood restaurante", k=5)

    assert [n.expense_id for n in found] == [mine]
    assert found[0].similarity > 0.99


def test_upsert_replaces_rows_in_place(index: PgTransactionIndex) -> None:
    expense_id = f"e-{uuid.uuid4().hex}"
    index.upsert("alice", [item(expense_id, "amazon", "shopping")])
    index.upsert("alice", [item(expense_id, "amazon", "books")])

    assert index.indexed([expense_id])[expense_id]["category_id"] == "books"
    assert len(index.search("alice", "expense", "amazon", k=5)) == 1


def test_lists_and_deletes_the_rows_of_one_user(index: PgTransactionIndex) -> None:
    kept, removed, theirs = (f"{prefix}-{uuid.uuid4().hex}" for prefix in "krb")
    index.upsert("alice", [item(kept, "uber trip", "transport"), item(removed, "padaria", "food")])
    index.upsert("bob", [item(theirs, "padaria", "food")])

    index.delete([removed])

    assert index.indexed_ids("alice") == {kept}
    assert index.indexed_ids("bob") == {theirs}
