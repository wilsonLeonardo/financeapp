"""Async client for the FinanceApp Go API, acting with the caller's token.

The Go API stays the source of truth and the authority on access: every call carries the user's own
JWT, so the agents' tools can only ever read or change that user's data.
"""

from datetime import date
from decimal import Decimal
from typing import Any

import httpx
from pydantic import BaseModel

UNCATEGORIZED = "none"  # mirrors expense.UncategorizedFilter in the Go API
MAX_PAGE_SIZE = 100  # the Go API caps page_size at 100


class Category(BaseModel):
    id: str
    name: str


class Expense(BaseModel):
    id: str
    description: str
    amount: Decimal
    type: str
    date: str
    category_id: str | None = None
    tags: str = ""

    @property
    def day(self) -> str:
        return self.date[:10]


class MonthlySummary(BaseModel):
    month: str
    total_spent: float
    total_earned: float


class CategorySummary(BaseModel):
    category_id: str | None = None
    category_name: str
    total: float
    count: int


class FinanceAPIError(Exception):
    def __init__(self, status_code: int, message: str):
        super().__init__(f"{status_code}: {message}")
        self.status_code = status_code
        self.message = message


class FinanceAPI:
    """Wraps a shared ``httpx.AsyncClient`` (base URL set) with one user's token."""

    def __init__(self, client: httpx.AsyncClient, token: str):
        self._client = client
        self._headers = {"Authorization": f"Bearer {token}"}

    async def categories(self) -> list[Category]:
        return [Category.model_validate(c) for c in await self._get("/categories")]

    async def expenses_page(
        self,
        *,
        page: int = 1,
        page_size: int = MAX_PAGE_SIZE,
        category_id: str | None = None,
        type: str | None = None,
        start_date: date | None = None,
        end_date: date | None = None,
    ) -> tuple[list[Expense], int]:
        params: dict[str, Any] = {"page": page, "page_size": page_size}
        if category_id:
            params["category_id"] = category_id
        if type:
            params["type"] = type
        if start_date:
            params["start_date"] = start_date.isoformat()
        if end_date:
            params["end_date"] = end_date.isoformat()
        body = await self._get("/expenses", params)
        return [Expense.model_validate(e) for e in body["data"]], body["total"]

    async def expenses(self, *, limit: int = 1000, **filters: Any) -> list[Expense]:
        """Pages through /expenses until ``limit`` rows or the end of the result set."""
        rows: list[Expense] = []
        page = 1
        while len(rows) < limit:
            batch, total = await self.expenses_page(page=page, **filters)
            rows.extend(batch)
            if not batch or len(rows) >= total:
                break
            page += 1
        return rows[:limit]

    async def get_expense(self, expense_id: str) -> Expense:
        return Expense.model_validate(await self._get(f"/expenses/{expense_id}"))

    async def set_category(self, expense_id: str, category_id: str) -> Expense:
        # PUT /expenses/:id replaces tags with whatever the body carries, so the current tags are
        # sent back; otherwise categorizing a transaction would silently wipe them.
        current = await self.get_expense(expense_id)
        resp = await self._client.put(
            f"/expenses/{expense_id}",
            json={"category_id": category_id, "tags": current.tags},
            headers=self._headers,
        )
        return Expense.model_validate(self._json(resp))

    async def monthly_summary(self) -> list[MonthlySummary]:
        return [MonthlySummary.model_validate(m) for m in await self._get("/reports/monthly")]

    async def category_summary(self, start: date, end: date) -> list[CategorySummary]:
        rows = await self._get(
            "/reports/categories", {"start_date": start.isoformat(), "end_date": end.isoformat()}
        )
        return [CategorySummary.model_validate(c) for c in rows]

    async def _get(self, path: str, params: dict[str, Any] | None = None) -> Any:
        resp = await self._client.get(path, params=params, headers=self._headers)
        return self._json(resp)

    @staticmethod
    def _json(resp: httpx.Response) -> Any:
        if resp.is_error:
            try:
                message = resp.json().get("message", resp.text)
            except ValueError:
                message = resp.text
            raise FinanceAPIError(resp.status_code, message)
        return resp.json()
