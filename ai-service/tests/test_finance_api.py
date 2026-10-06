import json
from datetime import date

import httpx
import pytest
import respx

from app.finance_api import UNCATEGORIZED, FinanceAPI, FinanceAPIError
from tests.conftest import API


def expense(i: int, **overrides) -> dict:
    return {
        "id": f"e{i}",
        "description": f"item {i}",
        "amount": "10.50",
        "type": "expense",
        "date": "2026-09-10T00:00:00Z",
        "tags": "",
        **overrides,
    }


@pytest.fixture
async def api():
    async with httpx.AsyncClient(base_url=API) as client:
        yield FinanceAPI(client, "tok")


@respx.mock
async def test_pages_until_the_total_and_sends_filters(api: FinanceAPI) -> None:
    route = respx.get(f"{API}/expenses").mock(
        side_effect=[
            httpx.Response(
                200, json={"data": [expense(1), expense(2)], "total": 3, "page": 1, "page_size": 2}
            ),
            httpx.Response(200, json={"data": [expense(3)], "total": 3, "page": 2, "page_size": 2}),
        ]
    )

    rows = await api.expenses(page_size=2, category_id=UNCATEGORIZED, start_date=date(2026, 9, 1))

    assert [r.id for r in rows] == ["e1", "e2", "e3"]
    first = route.calls[0].request
    assert first.headers["Authorization"] == "Bearer tok"
    assert first.url.params["category_id"] == "none"
    assert first.url.params["start_date"] == "2026-09-01"


@respx.mock
async def test_setting_a_category_preserves_existing_tags(api: FinanceAPI) -> None:
    respx.get(f"{API}/expenses/e1").mock(return_value=httpx.Response(200, json=expense(1, tags="trip,work")))
    put = respx.put(f"{API}/expenses/e1").mock(
        return_value=httpx.Response(200, json=expense(1, tags="trip,work", category_id="c1"))
    )

    updated = await api.set_category("e1", "c1")

    assert json.loads(put.calls[0].request.content) == {"category_id": "c1", "tags": "trip,work"}
    assert updated.category_id == "c1"


@respx.mock
async def test_api_errors_carry_status_and_message(api: FinanceAPI) -> None:
    respx.get(f"{API}/categories").mock(return_value=httpx.Response(401, json={"message": "token revoked"}))

    with pytest.raises(FinanceAPIError) as exc:
        await api.categories()
    assert (exc.value.status_code, exc.value.message) == (401, "token revoked")
