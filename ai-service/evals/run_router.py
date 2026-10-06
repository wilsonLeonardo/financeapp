"""Measures how often the router hands a message to the right specialist.

Usage: python -m evals.run_router
Needs Ollama with the configured chat model.
"""

import asyncio
import json
import time
from collections import Counter
from datetime import UTC, datetime
from pathlib import Path

import pandas as pd

from app.agents.graph import ROUTES, choose_route
from app.config import get_settings
from app.llm import chat_model

DATA = Path(__file__).parent / "data" / "router_prompts.jsonl"
RESULTS = Path(__file__).parent / "results" / "router.json"


async def main() -> None:
    settings = get_settings()
    rows = [json.loads(line) for line in DATA.read_text().splitlines() if line.strip()]
    llm = chat_model(settings)
    await llm.ainvoke("ok")  # loads the model before timing

    predictions, latencies = [], []
    for row in rows:
        started = time.perf_counter()
        predictions.append(await choose_route(llm, row["prompt"]))
        latencies.append((time.perf_counter() - started) * 1000)

    pairs = [(r["route"], p) for r, p in zip(rows, predictions, strict=True)]
    totals = Counter(t for t, _ in pairs)
    hits = Counter(t for t, p in pairs if t == p)
    accuracy = sum(hits.values()) / len(pairs)

    print(
        pd.DataFrame(
            [{"route": r, "prompts": totals[r], "recall": f"{hits[r] / totals[r]:.0%}"} for r in ROUTES]
        ).to_markdown(index=False)
    )
    median = sorted(latencies)[len(latencies) // 2]
    print(f"\naccuracy: {accuracy:.1%} on {len(pairs)} prompts, median {median:.0f} ms")
    misroutes = [
        {"prompt": r["prompt"], "expected": t, "got": p}
        for r, (t, p) in zip(rows, pairs, strict=True)
        if t != p
    ]
    for m in misroutes:
        print(f"  misrouted: {m['prompt']!r} -> {m['got']} (expected {m['expected']})")

    RESULTS.parent.mkdir(parents=True, exist_ok=True)
    RESULTS.write_text(
        json.dumps(
            {
                "generated_at": datetime.now(UTC).isoformat(timespec="seconds"),
                "chat_model": settings.chat_model,
                "prompts": len(pairs),
                "accuracy": accuracy,
                "recall": {r: hits[r] / totals[r] for r in ROUTES},
                "latency_ms_median": median,
                "misroutes": misroutes,
            },
            ensure_ascii=False,
            indent=2,
        )
        + "\n"
    )
    print(f"saved {RESULTS}")


if __name__ == "__main__":
    asyncio.run(main())
