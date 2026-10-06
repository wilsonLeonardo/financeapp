"""Tools the agents call. Each set is built per request around the caller's FinanceAPI client.

Design rules, all aimed at small local models:
- Few tools with short, explicit argument schemas.
- Tools do the arithmetic (totals, counts, nets) so the model only has to read numbers out.
- Results are capped and compact: aggregates and a handful of rows, never a full dump.
- Read-only. Applying categories happens in the review UI, after the user confirms.
- Failures come back as {"error": ...} so the model can report them instead of guessing.
"""

import difflib
import unicodedata
from calendar import monthrange
from collections.abc import Awaitable, Callable
from datetime import date
from decimal import Decimal
from functools import wraps
from typing import Any, Literal

from langchain_core.tools import BaseTool, StructuredTool
from pydantic import BaseModel, Field, field_validator

from app.finance_api import UNCATEGORIZED, Category, FinanceAPI, FinanceAPIError
from app.rag.categorizer import Categorizer, Transaction
from app.rag.normalize import normalize_description
from app.rag.sync import categorized_history, category_exemplars, sync_index

SEARCH_SCAN_LIMIT = 500  # rows fetched per search; enough for months of personal transactions


def resolve_period(
    month: str | None, start_date: str | None, end_date: str | None, today: date
) -> tuple[date, date]:
    """Turns the period arguments into dates; defaults to the current month.

    Months are whole calendar months, future-dated entries included: users log planned expenses
    ahead, and "this month" means all of them. Small models often pass only one end of a range,
    typically today's date for "this month", so a lone end_date means that date's whole month and a
    lone start_date runs to the end of the current month. Only an explicit range is cut mid-month.
    """
    if month:
        year, mon = (int(part) for part in month.split("-"))
        start, end = _month_start(date(year, mon, 1)), _month_end(date(year, mon, 1))
    elif start_date and end_date:
        start, end = date.fromisoformat(start_date), date.fromisoformat(end_date)
    elif end_date:
        start, end = _month_start(date.fromisoformat(end_date)), _month_end(date.fromisoformat(end_date))
    elif start_date:
        start, end = date.fromisoformat(start_date), _month_end(today)
    else:
        start, end = _month_start(today), _month_end(today)
    if start > end:
        raise ValueError("start_date is after end_date; for a calendar month pass month='YYYY-MM'")
    return start, end


def _month_start(day: date) -> date:
    return day.replace(day=1)


def _month_end(day: date) -> date:
    return day.replace(day=monthrange(day.year, day.month)[1])


def _money(value: Decimal | float) -> str:
    """Amounts leave the tools already formatted. A small model copies "$13,015.75" reliably but
    mangles thousands separators ("$1,3015.75") when it formats a raw number itself."""
    amount = round(float(value), 2)
    return f"-${-amount:,.2f}" if amount < 0 else f"${amount:,.2f}"


def _reporting_errors(fn: Callable[..., Awaitable[dict]]) -> Callable[..., Awaitable[dict]]:
    @wraps(fn)
    async def wrapper(**kwargs: Any) -> dict:
        try:
            return await fn(**kwargs)
        except ValueError as exc:
            return {"error": f"invalid argument: {exc}"}
        except FinanceAPIError as exc:
            return {"error": f"FinanceApp API error: {exc.message}"}

    return wrapper


class PeriodArgs(BaseModel):
    month: str | None = Field(
        None, description='Calendar month as "YYYY-MM". Preferred for monthly questions.'
    )
    start_date: str | None = Field(None, description="Period start as YYYY-MM-DD, when not using month.")
    end_date: str | None = Field(None, description="Period end as YYYY-MM-DD, when not using month.")


class CategoryTotalsArgs(PeriodArgs):
    category: str | None = Field(
        None, description="Optional: one category to detail, which returns its transactions instead."
    )


class MonthlyArgs(PeriodArgs):
    last_n_months: int = Field(
        6, ge=1, le=12, description="How many recent months to return when no month or period is given."
    )


class SearchArgs(PeriodArgs):
    category: str | None = Field(
        None, description='Category name, or "uncategorized". A merchant name is searched in descriptions.'
    )
    type: Literal["expense", "income"] | None = Field(None, description="Only expenses or only income.")
    text: str | None = Field(None, description='Words the description must contain, e.g. "uber" or "ifood".')
    limit: int = Field(10, ge=1, le=20, description="How many of the newest matching transactions to list.")

    @field_validator("type", mode="before")
    @classmethod
    def _plain_type(cls, value: object) -> object:
        # Small models write "expenses" or the Portuguese word; map those instead of failing the call.
        if isinstance(value, str):
            return _TYPE_ALIASES.get(_fold(value), value)
        return value


class SuggestArgs(BaseModel):
    limit: int = Field(10, ge=1, le=30, description="How many uncategorized transactions to suggest for.")


def analyst_tools(api: FinanceAPI, today: date) -> list[BaseTool]:
    """The analyst's tools. They accept the arguments a small model reaches for first, even on the
    "wrong" tool: totals asked for one category return its transactions, a merchant passed as a
    category is searched in descriptions, and every tool takes a period."""

    async def list_categories() -> dict:
        return {"categories": [c.name for c in await api.categories()]}

    async def search_transactions(
        month: str | None = None,
        start_date: str | None = None,
        end_date: str | None = None,
        category: str | None = None,
        type: Literal["expense", "income"] | None = None,
        text: str | None = None,
        limit: int = 10,
    ) -> dict:
        filters: dict[str, Any] = {"type": type}
        if month or start_date or end_date:
            filters["start_date"], filters["end_date"] = resolve_period(month, start_date, end_date, today)
        note = None
        if category:
            category_id = await _category_id(api, category)
            if category_id:
                filters["category_id"] = category_id
            else:
                text = f"{text} {category}" if text else category
                note = f'"{category}" is not a category, so descriptions containing it were searched.'

        rows = await api.expenses(limit=SEARCH_SCAN_LIMIT, **filters)
        if text:
            needle = normalize_description(text)
            rows = [r for r in rows if needle in normalize_description(r.description)]

        names = {c.id: c.name for c in await api.categories()} if rows else {}
        result: dict[str, Any] = {
            "count": len(rows),
            "total": _money(sum((r.amount for r in rows), Decimal(0))),
            "truncated": len(rows) >= SEARCH_SCAN_LIMIT,
            "transactions": [
                {
                    "date": r.day,
                    "description": r.description,
                    "amount": _money(r.amount),
                    "type": r.type,
                    "category": names.get(r.category_id or "", "Uncategorized"),
                }
                for r in rows[:limit]
            ],
        }
        if note:
            result["note"] = note
        return result

    async def spending_by_category(
        month: str | None = None,
        start_date: str | None = None,
        end_date: str | None = None,
        category: str | None = None,
    ) -> dict:
        start, end = resolve_period(month, start_date, end_date, today)
        period = {"start": start.isoformat(), "end": end.isoformat()}
        if category:  # the model wants one category's details: give it the transactions
            detail = await search_transactions(
                start_date=period["start"], end_date=period["end"], category=category, type="expense"
            )
            return {"period": period, "category": category, **detail}

        rows = await api.category_summary(start, end)
        return {
            "period": period,
            "total_spent": _money(sum(r.total for r in rows)),
            "categories": [
                {"category": r.category_name, "total": _money(r.total), "transactions": r.count} for r in rows
            ],
        }

    async def monthly_totals(
        month: str | None = None,
        start_date: str | None = None,
        end_date: str | None = None,
        last_n_months: int = 6,
    ) -> dict:
        months = await api.monthly_summary()
        if month or start_date or end_date:
            start, end = resolve_period(month, start_date, end_date, today)
            months = [m for m in months if f"{start:%Y-%m}" <= m.month <= f"{end:%Y-%m}"]
        else:
            months = months[-last_n_months:]
        return {
            "months": [
                {
                    "month": m.month,
                    "spent": _money(m.total_spent),
                    "earned": _money(m.total_earned),
                    "net": _money(m.total_earned - m.total_spent),
                }
                for m in months
            ]
        }

    return [
        StructuredTool.from_function(
            coroutine=_reporting_errors(list_categories),
            name="list_categories",
            description="List the user's category names.",
        ),
        StructuredTool.from_function(
            coroutine=_reporting_errors(spending_by_category),
            name="spending_by_category",
            description=(
                "Total spent per category in a period (expenses only), plus the overall total. Totals "
                "only: to see the transactions inside a category, use search_transactions."
            ),
            args_schema=CategoryTotalsArgs,
        ),
        StructuredTool.from_function(
            coroutine=_reporting_errors(monthly_totals),
            name="monthly_totals",
            description=(
                "Spent, earned and net per month, for a given month or period or the most recent months."
            ),
            args_schema=MonthlyArgs,
        ),
        StructuredTool.from_function(
            coroutine=_reporting_errors(search_transactions),
            name="search_transactions",
            description=(
                "List individual transactions, newest first, with their count and total. Use it to detail a "
                'category ("details of Casa"), find a merchant ("uber", "netflix") or list a period.'
            ),
            args_schema=SearchArgs,
        ),
    ]


def categorizer_tools(api: FinanceAPI, categorizer: Categorizer | None, user_id: str) -> list[BaseTool]:
    async def suggest_categories(limit: int = 10) -> dict:
        if categorizer is None:
            return {"error": "categorization is unavailable right now (vector index offline)"}
        pending, total = await api.expenses_page(page_size=limit, category_id=UNCATEGORIZED)
        if not pending:
            return {"uncategorized_total": 0, "suggestions": []}

        history = await categorized_history(api)
        await sync_index(categorizer.index, user_id, history)
        categories = await api.categories()
        names = {c.id: c.name for c in categories}
        txns = [Transaction(e.id, e.description, e.type, str(e.amount)) for e in pending]
        suggestions = await categorizer.categorize_many(
            user_id, txns, categories, category_exemplars(history)
        )
        return {
            "uncategorized_total": total,
            "suggestions": [
                {
                    "description": e.description,
                    "amount": _money(e.amount),
                    "suggested_category": names.get(s.category_id or ""),
                    "source": {"knn": "similar past transactions", "llm": "model", "none": "no suggestion"}[
                        s.method
                    ],
                }
                for e, s in zip(pending, suggestions, strict=True)
            ],
            "next_step": "The user reviews and applies suggestions on the Transactions page.",
        }

    return [
        StructuredTool.from_function(
            coroutine=_reporting_errors(suggest_categories),
            name="suggest_categories",
            description=(
                "Suggest categories for the user's uncategorized transactions, using their categorization "
                "history. Does not change any data."
            ),
            args_schema=SuggestArgs,
        )
    ]


_TYPE_ALIASES = {
    **dict.fromkeys(["expense", "expenses", "spending", "despesa", "despesas", "gasto", "gastos"], "expense"),
    **dict.fromkeys(["income", "incomes", "earnings", "receita", "receitas", "renda"], "income"),
}


def _fold(text: str) -> str:
    return unicodedata.normalize("NFKD", text).encode("ascii", "ignore").decode().casefold().strip()


async def _category_id(api: FinanceAPI, name: str) -> str | None:
    """The category a name refers to, or None when it is not one (then it is likely a merchant)."""
    wanted = _fold(name)
    if wanted in {"uncategorized", "sem categoria", "none"}:
        return UNCATEGORIZED
    categories: list[Category] = await api.categories()
    by_name = {_fold(c.name): c.id for c in categories}
    if wanted in by_name:
        return by_name[wanted]
    # Small models pluralize or misspell names ("Transportes"): accept one clearly close match.
    if close := difflib.get_close_matches(wanted, list(by_name), n=1, cutoff=0.8):
        return by_name[close[0]]
    return None
