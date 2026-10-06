"""Vector index of a user's already-categorized transactions.

Production uses pgvector (the Postgres the Go API already runs); evals and tests use an in-memory
store. Both sit behind the same interface so the categorizer is exercised identically in each.

Every search is scoped to one user. Without that filter a nearest-neighbour lookup would happily
return other people's transactions, so it is enforced here rather than left to callers.
"""

from abc import ABC, abstractmethod
from collections.abc import Sequence
from dataclasses import dataclass

from langchain_core.documents import Document
from langchain_core.embeddings import Embeddings
from langchain_core.vectorstores import InMemoryVectorStore
from langchain_postgres import PGVector


@dataclass(frozen=True)
class IndexedTransaction:
    expense_id: str
    text: str  # normalize.index_text(...)
    txn_type: str
    category_id: str


@dataclass(frozen=True)
class Neighbor:
    expense_id: str
    category_id: str
    text: str
    similarity: float  # cosine similarity, 1.0 = identical


def _document(user_id: str, item: IndexedTransaction, model: str) -> Document:
    return Document(
        id=item.expense_id,
        page_content=item.text,
        metadata={
            "user_id": user_id,
            "expense_id": item.expense_id,
            "type": item.txn_type,
            "category_id": item.category_id,
            "text": item.text,  # lets a sync notice descriptions or normalization that changed
            "model": model,  # and a change of embedding model
        },
    )


def _neighbor(doc: Document, similarity: float) -> Neighbor:
    return Neighbor(
        expense_id=doc.metadata["expense_id"],
        category_id=doc.metadata["category_id"],
        text=doc.page_content,
        similarity=similarity,
    )


class TransactionIndex(ABC):
    model: str  # embedding model the stored vectors came from

    @abstractmethod
    def upsert(self, user_id: str, items: Sequence[IndexedTransaction]) -> None: ...

    @abstractmethod
    def search(self, user_id: str, txn_type: str, text: str, k: int) -> list[Neighbor]:
        """The ``k`` most similar transactions of this user and type, most similar first."""

    @abstractmethod
    def indexed(self, expense_ids: Sequence[str]) -> dict[str, dict]:
        """Metadata of the given IDs that are already indexed, keyed by expense ID."""

    @abstractmethod
    def indexed_ids(self, user_id: str) -> set[str]:
        """IDs of every transaction indexed for this user."""

    @abstractmethod
    def delete(self, expense_ids: Sequence[str]) -> None: ...


class PgTransactionIndex(TransactionIndex):
    def __init__(self, embeddings: Embeddings, connection: str, collection: str, model: str):
        # PGVector creates the vector extension and its tables on first use, and upserts by ID.
        # IDs are unique across collections, so one collection serves every embedding model: a
        # model change re-embeds each row in place (see sync) instead of starting a new collection.
        self.model = model
        self.store = PGVector(
            embeddings=embeddings, connection=connection, collection_name=collection, use_jsonb=True
        )

    def upsert(self, user_id: str, items: Sequence[IndexedTransaction]) -> None:
        if items:
            self.store.add_documents(
                [_document(user_id, i, self.model) for i in items], ids=[i.expense_id for i in items]
            )

    def search(self, user_id: str, txn_type: str, text: str, k: int) -> list[Neighbor]:
        results = self.store.similarity_search_with_score(
            text, k=k, filter={"user_id": {"$eq": user_id}, "type": {"$eq": txn_type}}
        )
        # PGVector's default strategy returns cosine distance.
        return [_neighbor(doc, 1.0 - distance) for doc, distance in results]

    def indexed(self, expense_ids: Sequence[str]) -> dict[str, dict]:
        if not expense_ids:
            return {}
        return {doc.metadata["expense_id"]: doc.metadata for doc in self.store.get_by_ids(list(expense_ids))}

    def indexed_ids(self, user_id: str) -> set[str]:
        # PGVector has no "list by metadata", so this queries its table; the containment filter (@>)
        # uses the GIN index PGVector keeps on the metadata.
        rows = self.store.EmbeddingStore
        with self.store.session_maker() as session:
            collection = self.store.get_collection(session)
            if collection is None:
                return set()
            query = session.query(rows.id).filter(
                rows.collection_id == collection.uuid, rows.cmetadata.contains({"user_id": user_id})
            )
            return {expense_id for (expense_id,) in query}

    def delete(self, expense_ids: Sequence[str]) -> None:
        if expense_ids:
            self.store.delete(list(expense_ids), collection_only=True)


class MemoryTransactionIndex(TransactionIndex):
    def __init__(self, embeddings: Embeddings, model: str = "memory"):
        self.model = model
        self.store = InMemoryVectorStore(embeddings)

    def upsert(self, user_id: str, items: Sequence[IndexedTransaction]) -> None:
        if items:
            self.store.add_documents(
                [_document(user_id, i, self.model) for i in items], ids=[i.expense_id for i in items]
            )

    def search(self, user_id: str, txn_type: str, text: str, k: int) -> list[Neighbor]:
        def same_user_and_type(doc: Document) -> bool:
            return doc.metadata["user_id"] == user_id and doc.metadata["type"] == txn_type

        results = self.store.similarity_search_with_score(text, k=k, filter=same_user_and_type)
        return [_neighbor(doc, score) for doc, score in results]

    def indexed(self, expense_ids: Sequence[str]) -> dict[str, dict]:
        return {doc.metadata["expense_id"]: doc.metadata for doc in self.store.get_by_ids(list(expense_ids))}

    def indexed_ids(self, user_id: str) -> set[str]:
        return {doc_id for doc_id, doc in self.store.store.items() if doc["metadata"]["user_id"] == user_id}

    def delete(self, expense_ids: Sequence[str]) -> None:
        self.store.delete(list(expense_ids))
