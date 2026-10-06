"""Measures categorization accuracy of the three strategies against the local models.

Time-based split, like a new statement arriving: the first months are the user's history (indexed),
the last month is the test set. Nothing from the test month is ever indexed, so retrieval cannot
peek at the answers. Accuracy is also broken down by whether the merchant appears in the history.

Usage: python -m evals.run_categorization [--limit N] [--strategies rag,knn,zero_shot]
Needs Ollama with the configured chat and embedding models (see `make ai-models`).
"""

import argparse
import asyncio
import json
import statistics
import time
from datetime import UTC, datetime
from pathlib import Path

import pandas as pd

from app.config import get_settings
from app.finance_api import Category, Expense
from app.llm import chat_model, embeddings
from app.rag.categorizer import Categorizer, Strategy, Transaction
from app.rag.normalize import index_text
from app.rag.store import IndexedTransaction, MemoryTransactionIndex
from app.rag.sync import category_exemplars

DATA = Path(__file__).parent / "data" / "transactions.jsonl"
RESULTS = Path(__file__).parent / "results" / "categorization.json"
USER = "eval-user"


def load() -> tuple[list[dict], list[dict]]:
    rows = [json.loads(line) for line in DATA.read_text().splitlines() if line.strip()]
    last_month = max(r["date"][:7] for r in rows)
    return [r for r in rows if r["date"][:7] < last_month], [r for r in rows if r["date"][:7] == last_month]


def macro_f1(truth: list[str], pred: list[str | None], labels: list[str]) -> float:
    scores = []
    for label in labels:
        tp = sum(t == label and p == label for t, p in zip(truth, pred, strict=True))
        fp = sum(t != label and p == label for t, p in zip(truth, pred, strict=True))
        fn = sum(t == label and p != label for t, p in zip(truth, pred, strict=True))
        precision = tp / (tp + fp) if tp + fp else 0.0
        recall = tp / (tp + fn) if tp + fn else 0.0
        scores.append(2 * precision * recall / (precision + recall) if precision + recall else 0.0)
    return sum(scores) / len(scores)


def accuracy(pairs: list[tuple[str, str | None]]) -> float | None:
    return sum(t == p for t, p in pairs) / len(pairs) if pairs else None


async def evaluate(
    strategy: Strategy,
    categorizer: Categorizer,
    test: list[dict],
    seen: set[str],
    categories: list[Category],
    exemplars: dict[str, list[str]],
) -> dict:
    name_of = {c.id: c.name for c in categories}
    preds: list[str | None] = []
    methods: list[str] = []
    latencies: list[float] = []
    for row in test:
        txn = Transaction(row["id"], row["description"], row["type"], str(row["amount"]))
        started = time.perf_counter()
        suggestion = await categorizer.categorize(USER, txn, categories, exemplars)
        latencies.append((time.perf_counter() - started) * 1000)
        preds.append(name_of.get(suggestion.category_id or ""))
        methods.append(suggestion.method)

    truth = [r["category"] for r in test]
    pairs = list(zip(truth, preds, strict=True))
    known = [p for p, r in zip(pairs, test, strict=True) if r["merchant"] in seen]
    new = [p for p, r in zip(pairs, test, strict=True) if r["merchant"] not in seen]
    return {
        "strategy": strategy.value,
        "accuracy": accuracy(pairs),
        "accuracy_seen_merchants": accuracy(known),
        "accuracy_new_merchants": accuracy(new),
        "macro_f1": macro_f1(truth, preds, sorted({c.name for c in categories})),
        "resolved_without_llm": methods.count("knn") / len(methods),
        "no_suggestion": preds.count(None) / len(preds),
        "latency_ms_p50": statistics.median(latencies),
        "latency_ms_p95": sorted(latencies)[max(0, int(len(latencies) * 0.95) - 1)],
        "errors": [
            {"description": r["description"], "expected": t, "predicted": p, "method": m}
            for r, (t, p), m in zip(test, pairs, methods, strict=True)
            if t != p
        ],
    }


async def main(limit: int | None, strategies: list[Strategy]) -> None:
    settings = get_settings()
    history, test = load()
    test = test[:limit] if limit else test
    seen = {r["merchant"] for r in history}
    categories = [
        Category(id=f"c{i}", name=n) for i, n in enumerate(sorted({r["category"] for r in history}))
    ]
    id_of = {c.name: c.id for c in categories}

    llm = chat_model(settings)
    index = MemoryTransactionIndex(embeddings(settings), settings.embed_model)
    print(f"indexing {len(history)} history transactions with {settings.embed_model}...")
    await asyncio.to_thread(
        index.upsert,
        USER,
        [
            IndexedTransaction(r["id"], index_text(r["description"]), r["type"], id_of[r["category"]])
            for r in history
        ],
    )
    exemplars = category_exemplars(
        [
            Expense(id=r["id"], description=r["description"], amount=r["amount"], type=r["type"],
                    date=r["date"], category_id=id_of[r["category"]])
            for r in history
        ]
    )  # fmt: skip
    await llm.ainvoke("ok")  # loads the model so the first timed call is not a cold start

    results = []
    for strategy in strategies:
        print(f"running {strategy.value} on {len(test)} transactions...")
        categorizer = Categorizer(
            llm, index, k=settings.knn_k, auto_threshold=settings.knn_auto_threshold, strategy=strategy
        )
        results.append(await evaluate(strategy, categorizer, test, seen, categories, exemplars))

    table = pd.DataFrame(
        [
            {
                "strategy": r["strategy"],
                "accuracy": f"{r['accuracy']:.1%}",
                "seen merchants": f"{r['accuracy_seen_merchants']:.1%}",
                "new merchants": f"{r['accuracy_new_merchants']:.1%}",
                "macro F1": f"{r['macro_f1']:.3f}",
                "no LLM call": f"{r['resolved_without_llm']:.0%}",
                "p50 ms": round(r["latency_ms_p50"]),
                "p95 ms": round(r["latency_ms_p95"]),
            }
            for r in results
        ]
    )
    print()
    print(table.to_markdown(index=False))

    RESULTS.parent.mkdir(parents=True, exist_ok=True)
    RESULTS.write_text(
        json.dumps(
            {
                "generated_at": datetime.now(UTC).isoformat(timespec="seconds"),
                "chat_model": settings.chat_model,
                "embed_model": settings.embed_model,
                "k": settings.knn_k,
                "auto_threshold": settings.knn_auto_threshold,
                "history_transactions": len(history),
                "test_transactions": len(test),
                "test_from_new_merchants": sum(r["merchant"] not in seen for r in test),
                "results": results,
            },
            ensure_ascii=False,
            indent=2,
        )
        + "\n"
    )
    print(f"\nsaved {RESULTS}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--limit", type=int, help="evaluate only the first N test transactions")
    parser.add_argument("--strategies", default="zero_shot,knn,rag", help="comma-separated strategies")
    args = parser.parse_args()
    asyncio.run(main(args.limit, [Strategy(s) for s in args.strategies.split(",")]))
