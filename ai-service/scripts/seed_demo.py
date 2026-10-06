"""Fills a running FinanceApp stack with a demo account built from the synthetic dataset.

The first months become already-categorized transactions (the account's history). The last month is
added without categories, so the assistant and the AI suggestions on the Transactions page have
something to work on. Safe to re-run: an account that already has transactions is left alone.

Usage: python -m scripts.seed_demo [--api URL] [--email EMAIL] [--password PASSWORD]
"""

import argparse
import json

import httpx

from evals.generate_dataset import OUT as DATASET


def login(client: httpx.Client, name: str, email: str, password: str) -> str:
    resp = client.post("/auth/register", json={"name": name, "email": email, "password": password})
    if resp.status_code == 409:  # already registered
        resp = client.post("/auth/login", json={"email": email, "password": password})
    resp.raise_for_status()
    return resp.json()["token"]


def main(api: str, name: str, email: str, password: str) -> None:
    rows = [json.loads(line) for line in DATASET.read_text().splitlines() if line.strip()]
    last_month = max(r["date"][:7] for r in rows)

    with httpx.Client(base_url=api, timeout=60.0) as client:
        client.headers["Authorization"] = f"Bearer {login(client, name, email, password)}"

        if client.get("/expenses", params={"page_size": 1}).json()["total"] > 0:
            print(f"{email} already has transactions; nothing to do")
            return

        category_ids = {c["name"]: c["id"] for c in client.get("/categories").json()}
        for category in sorted({r["category"] for r in rows} - category_ids.keys()):
            category_ids[category] = (
                client.post("/categories", json={"name": category}).raise_for_status().json()["id"]
            )

        for r in rows:
            payload = {
                "amount": r["amount"],
                "type": r["type"],
                "description": r["description"],
                "date": r["date"],
            }
            if r["date"][:7] < last_month:
                payload["category_id"] = category_ids[r["category"]]
            client.post("/expenses", json=payload).raise_for_status()

    uncategorized = sum(r["date"][:7] == last_month for r in rows)
    print(f"seeded {email}: {len(rows) - uncategorized} categorized, {uncategorized} left uncategorized")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--api", default="http://localhost:8080/api/v1")
    parser.add_argument("--name", default="Demo")
    parser.add_argument("--email", default="demo@financeapp.dev")
    parser.add_argument("--password", default="demo-password-123")
    args = parser.parse_args()
    main(args.api, args.name, args.email, args.password)
