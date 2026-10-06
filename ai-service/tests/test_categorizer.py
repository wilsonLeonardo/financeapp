from app.finance_api import Category
from app.rag.categorizer import UNKNOWN, Categorizer, Strategy, Transaction
from app.rag.normalize import index_text
from app.rag.store import IndexedTransaction, MemoryTransactionIndex
from tests.conftest import FakeStructuredLLM

FOOD = Category(id="c-food", name="Alimentação")
TRANSPORT = Category(id="c-transport", name="Transporte")
CATEGORIES = [FOOD, TRANSPORT]


def seeded_index(embeddings, rows: list[tuple[str, str, str]]) -> MemoryTransactionIndex:
    index = MemoryTransactionIndex(embeddings)
    index.upsert(
        "u1",
        [IndexedTransaction(eid, index_text(desc), "expense", cat) for eid, desc, cat in rows],
    )
    return index


def txn(description: str) -> Transaction:
    return Transaction("t1", description, "expense", "42.00")


async def test_near_identical_agreeing_neighbours_skip_the_llm(embeddings) -> None:
    index = seeded_index(
        embeddings,
        [("e1", "IFOOD *RESTAURANTE 01/09", "c-food"), ("e2", "COMPRA IFOOD *RESTAURANTE 4321", "c-food")],
    )
    llm = FakeStructuredLLM([])

    result = await Categorizer(llm, index).categorize("u1", txn("IFOOD*RESTAURANTE 15/10"), CATEGORIES)

    assert (result.category_id, result.method) == ("c-food", "knn")
    assert result.confidence and result.confidence > 0.99
    assert llm.calls == []


async def test_falls_back_to_the_llm_with_neighbours_as_examples(embeddings) -> None:
    index = seeded_index(embeddings, [("e1", "UBER TRIP", "c-transport")])
    llm = FakeStructuredLLM([{"category": "transporte"}])

    result = await Categorizer(llm, index).categorize("u1", txn("99 POP CORRIDA"), CATEGORIES)

    assert (result.category_id, result.method) == ("c-transport", "llm")
    prompt = llm.calls[0][1].content
    assert '"uber trip" -> Transporte' in prompt
    assert llm.schemas[0]["properties"]["category"]["enum"] == ["Alimentação", "Transporte", UNKNOWN]


async def test_unknown_or_failed_llm_answers_leave_the_category_empty(embeddings) -> None:
    index = seeded_index(embeddings, [])
    llm = FakeStructuredLLM([{"category": UNKNOWN}, RuntimeError("ollama down")])
    categorizer = Categorizer(llm, index)

    for _ in range(2):
        result = await categorizer.categorize("u1", txn("SOMETHING ODD"), CATEGORIES)
        assert (result.category_id, result.method) == (None, "none")


async def test_neighbours_in_deleted_categories_are_ignored(embeddings) -> None:
    index = seeded_index(embeddings, [("e1", "IFOOD RESTAURANTE", "c-deleted")])
    llm = FakeStructuredLLM([{"category": "Alimentação"}])

    result = await Categorizer(llm, index).categorize("u1", txn("IFOOD RESTAURANTE"), CATEGORIES)

    assert result.method == "llm"
    assert result.neighbors == []


async def test_zero_shot_strategy_never_retrieves(embeddings) -> None:
    index = seeded_index(embeddings, [("e1", "IFOOD RESTAURANTE", "c-food")])
    llm = FakeStructuredLLM([{"category": "Alimentação"}])

    result = await Categorizer(llm, index, strategy=Strategy.ZERO_SHOT).categorize(
        "u1", txn("IFOOD RESTAURANTE"), CATEGORIES
    )

    assert result.method == "llm"
    assert "Similar past transactions" not in llm.calls[0][1].content


async def test_knn_strategy_votes_without_the_llm(embeddings) -> None:
    index = seeded_index(
        embeddings,
        [
            ("e1", "POSTO SHELL", "c-transport"),
            ("e2", "POSTO IPIRANGA", "c-transport"),
            ("e3", "PADARIA", "c-food"),
        ],
    )
    llm = FakeStructuredLLM([])

    result = await Categorizer(llm, index, strategy=Strategy.KNN).categorize(
        "u1", txn("POSTO BR"), CATEGORIES
    )

    assert (result.category_id, result.method) == ("c-transport", "knn")
    assert llm.calls == []


async def test_rag_prompt_describes_categories_with_the_users_own_merchants(embeddings) -> None:
    index = seeded_index(embeddings, [])
    exemplars = {"c-food": ["ifood restaurante", "padaria"], "c-transport": ["uber trip"]}

    rag_llm = FakeStructuredLLM([{"category": "Alimentação"}])
    await Categorizer(rag_llm, index).categorize("u1", txn("CANTINA NONNO"), CATEGORIES, exemplars)
    zero_llm = FakeStructuredLLM([{"category": "Alimentação"}])
    await Categorizer(zero_llm, index, strategy=Strategy.ZERO_SHOT).categorize(
        "u1", txn("CANTINA NONNO"), CATEGORIES, exemplars
    )

    assert "- Alimentação (e.g. ifood restaurante, padaria)" in rag_llm.calls[0][1].content
    assert "- Transporte (e.g. uber trip)" in rag_llm.calls[0][1].content
    assert "e.g." not in zero_llm.calls[0][1].content  # the zero-shot baseline gets names only
