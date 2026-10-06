"""Measures whether the analyst picks the right tool, with the right arguments, without narrating.

Each case is one user turn, optionally after earlier turns (follow-ups such as "and the details of
that category?"). The analyst decides exactly as in the graph (app.agents.graph.decide, including
its one retry when no tool is called) and tools never run, so no API or database is needed: a wrong
tool here means a wrong or invented answer in the app.

Usage: python -m evals.run_tools    (needs Ollama with the configured chat model)
"""

import asyncio
import json
import time
import unicodedata
from datetime import UTC, date, datetime
from pathlib import Path

import httpx
from langchain_core.messages import AIMessage, HumanMessage

from app.agents import prompts
from app.agents.graph import decide
from app.agents.tools import analyst_tools
from app.config import get_settings
from app.finance_api import FinanceAPI
from app.llm import chat_model

DATA = Path(__file__).parent / "data" / "tool_prompts.jsonl"
RESULTS = Path(__file__).parent / "results" / "tools.json"
TODAY = date(2026, 10, 6)


def fold(text: object) -> str:
    return unicodedata.normalize("NFKD", str(text or "")).encode("ascii", "ignore").decode().casefold()


async def main() -> None:
    settings = get_settings()
    cases = [json.loads(line) for line in DATA.read_text().splitlines() if line.strip()]
    async with httpx.AsyncClient() as client:
        # The tools are only needed for their schemas; they are never executed.
        model = chat_model(settings).bind_tools(analyst_tools(FinanceAPI(client, "unused"), TODAY))
        await model.ainvoke("ok")  # loads the model before timing

        rows = []
        for case in cases:
            history = case.get("history", [])
            turns = [HumanMessage(t) if i % 2 == 0 else AIMessage(t) for i, t in enumerate(history)]
            system = prompts.analyst(TODAY, prompts.reply_language(case["prompt"]))
            started = time.perf_counter()
            reply = await decide(model, system, [*turns, HumanMessage(case["prompt"])], grounded=True)
            elapsed = (time.perf_counter() - started) * 1000

            call = reply.tool_calls[0] if reply.tool_calls else None
            tool_ok = bool(call) and call["name"] == case["tool"]
            args_ok = tool_ok and all(
                fold(v) in fold(call["args"].get(k)) for k, v in case.get("args", {}).items()
            )
            rows.append(
                {
                    "prompt": case["prompt"],
                    "follow_up": bool(history),
                    "expected": case["tool"],
                    "got": call["name"] if call else None,
                    "args": call["args"] if call else None,
                    "tool_ok": tool_ok,
                    "args_ok": args_ok,
                    "narrated": bool(call) and bool(str(reply.text).strip()),
                    "ms": elapsed,
                }
            )
            mark = "ok " if args_ok else "BAD"
            got = json.dumps(rows[-1]["args"], ensure_ascii=False)
            print(f"{mark} {case['prompt'][:60]:60s} -> {rows[-1]['got']} {got}")

    def share(key: str, subset: list[dict]) -> float:
        return sum(r[key] for r in subset) / len(subset)

    follow = [r for r in rows if r["follow_up"]]
    summary = {
        "tool_accuracy": share("tool_ok", rows),
        "tool_and_args_accuracy": share("args_ok", rows),
        "follow_up_accuracy": share("args_ok", follow),
        "narrated_tool_calls": share("narrated", rows),
        "latency_ms_median": sorted(r["ms"] for r in rows)[len(rows) // 2],
    }
    print(
        f"\nright tool {summary['tool_accuracy']:.0%} | right tool and arguments "
        f"{summary['tool_and_args_accuracy']:.0%} | follow-ups {summary['follow_up_accuracy']:.0%} "
        f"| narrated {summary['narrated_tool_calls']:.0%} | n={len(rows)}"
    )
    RESULTS.parent.mkdir(parents=True, exist_ok=True)
    RESULTS.write_text(
        json.dumps(
            {
                "generated_at": datetime.now(UTC).isoformat(timespec="seconds"),
                "chat_model": settings.chat_model,
                "cases": len(rows),
                **summary,
                "rows": rows,
            },
            ensure_ascii=False,
            indent=2,
            default=str,
        )
        + "\n"
    )
    print(f"saved {RESULTS}")


if __name__ == "__main__":
    asyncio.run(main())
