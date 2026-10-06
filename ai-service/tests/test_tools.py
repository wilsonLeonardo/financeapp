from datetime import date

import httpx
import pytest
import respx

from app.agents.tools import analyst_tools, categorizer_tools, resolve_period
from app.finance_api import FinanceAPI
from tests.conftest import API

TODAY = date(2026, 10, 5)
CATEGORIES = [{"id": "c-food", "name": "Alimentação"}, {"id": "c-transport", "name": "Transporte"}]


def test_resolve_period_handles_months_ranges_and_the_default() -> None:
    assert resolve_period("2026-02", None, None, TODAY) == (date(2026, 2, 1), date(2026, 2, 28))
    assert resolve_period(None, "2026-09-10", "2026-09-20", TODAY) == (date(2026, 9, 10), date(2026, 9, 20))
    assert resolve_period(None, None, None, TODAY) == (date(2026, 10, 1), date(2026, 10, 31))
    assert resolve_period(None, None, "2026-08-31", TODAY) == (date(2026, 8, 1), date(2026, 8, 31))
    # "this month" sent as end_date=today still covers the whole month, future-dated entries included
    assert resolve_period(None, None, TODAY.isoformat(), TODAY) == (date(2026, 10, 1), date(2026, 10, 31))
    assert resolve_period(None, "2026-06-15", None, TODAY) == (date(2026, 6, 15), date(2026, 10, 31))
    assert resolve_period(None, "2026-10-01", "2026-10-05", TODAY) == (date(2026, 10, 1), date(2026, 10, 5))
    with pytest.raises(ValueError):
        resolve_period(None, "2026-09-20", "2026-09-10", TODAY)


@pytest.fixture
async def tools():
    async with httpx.AsyncClient(base_url=API) as client:
        yield {t.name: t for t in analyst_tools(FinanceAPI(client, "tok"), TODAY)}


@respx.mock
async def test_spending_by_category_totals_the_month(tools) -> None:
    route = respx.get(f"{API}/reports/categories").mock(
        return_value=httpx.Response(
            200,
            json=[
                {"category_id": "c-food", "category_name": "Alimentação", "total": 300.456, "count": 4},
                {"category_id": None, "category_name": "Uncategorized", "total": 50, "count": 1},
            ],
        )
    )

    result = await tools["spending_by_category"].ainvoke({"month": "2026-09"})

    assert route.calls[0].request.url.params["start_date"] == "2026-09-01"
    assert route.calls[0].request.url.params["end_date"] == "2026-09-30"
    assert result["total_spent"] == "$350.46"
    assert result["categories"][0] == {"category": "Alimentação", "total": "$300.46", "transactions": 4}


@respx.mock
async def test_search_filters_by_category_and_words_and_totals_matches(tools) -> None:
    respx.get(f"{API}/categories").mock(return_value=httpx.Response(200, json=CATEGORIES))
    expenses = respx.get(f"{API}/expenses").mock(
        return_value=httpx.Response(
            200,
            json={
                "data": [
                    {"id": "1", "description": "UBER *TRIP", "amount": "20.00", "type": "expense",
                     "date": "2026-09-12T00:00:00Z", "category_id": "c-transport"},
                    {"id": "2", "description": "99 POP", "amount": "15.00", "type": "expense",
                     "date": "2026-09-11T00:00:00Z", "category_id": "c-transport"},
                    {"id": "3", "description": "UBER *TRIP", "amount": "30.50", "type": "expense",
                     "date": "2026-09-02T00:00:00Z", "category_id": "c-transport"},
                ],
                "total": 3, "page": 1, "page_size": 100,
            },
        )
    )  # fmt: skip

    result = await tools["search_transactions"].ainvoke(
        {"category": "transporte", "text": "uber", "month": "2026-09"}
    )

    assert expenses.calls[0].request.url.params["category_id"] == "c-transport"
    assert (result["count"], result["total"]) == (2, "$50.50")
    assert result["transactions"][0] == {
        "date": "2026-09-12",
        "description": "UBER *TRIP",
        "amount": "$20.00",
        "type": "expense",
        "category": "Transporte",
    }


@respx.mock
async def test_a_merchant_passed_as_category_is_searched_in_descriptions(tools) -> None:
    respx.get(f"{API}/categories").mock(return_value=httpx.Response(200, json=CATEGORIES))
    expenses = respx.get(f"{API}/expenses").mock(
        return_value=httpx.Response(
            200,
            json={
                "data": [
                    {"id": "1", "description": "AMAZON MKTPL", "amount": "40.00", "type": "expense",
                     "date": "2026-09-12T00:00:00Z", "category_id": "c-food"},
                    {"id": "2", "description": "PADARIA", "amount": "9.00", "type": "expense",
                     "date": "2026-09-11T00:00:00Z", "category_id": "c-food"},
                ],
                "total": 2, "page": 1, "page_size": 100,
            },
        )
    )  # fmt: skip

    result = await tools["search_transactions"].ainvoke({"category": "Amazon"})

    assert "category_id" not in expenses.calls[0].request.url.params
    assert (result["count"], result["total"]) == (1, "$40.00")
    assert "not a category" in result["note"]


@respx.mock
async def test_monthly_totals_keeps_the_latest_months_and_computes_net(tools) -> None:
    respx.get(f"{API}/reports/monthly").mock(
        return_value=httpx.Response(
            200,
            json=[
                {"month": f"2026-0{m}", "total_spent": 100.0 * m, "total_earned": 1000.0} for m in range(1, 7)
            ],
        )
    )

    result = await tools["monthly_totals"].ainvoke({"last_n_months": 2})

    assert [m["month"] for m in result["months"]] == ["2026-05", "2026-06"]
    assert result["months"][-1]["net"] == "$400.00"


async def test_suggest_reports_when_the_index_is_offline() -> None:
    async with httpx.AsyncClient(base_url=API) as client:
        (tool,) = categorizer_tools(FinanceAPI(client, "tok"), None, "u1")
        result = await tool.ainvoke({})
    assert "unavailable" in result["error"]


@respx.mock
async def test_search_tolerates_plural_types_and_near_category_names(tools) -> None:
    respx.get(f"{API}/categories").mock(return_value=httpx.Response(200, json=CATEGORIES))
    expenses = respx.get(f"{API}/expenses").mock(
        return_value=httpx.Response(200, json={"data": [], "total": 0, "page": 1, "page_size": 100})
    )

    result = await tools["search_transactions"].ainvoke({"category": "Transportes", "type": "expenses"})

    params = expenses.calls[0].request.url.params
    assert (params["category_id"], params["type"]) == ("c-transport", "expense")
    assert result["count"] == 0


def test_money_is_formatted_for_the_model_to_copy() -> None:
    from app.agents.tools import _money

    assert (_money(13015.75), _money(-400), _money(0.004)) == ("$13,015.75", "-$400.00", "$0.00")


@respx.mock
async def test_totals_asked_for_one_category_return_its_transactions(tools) -> None:
    respx.get(f"{API}/categories").mock(return_value=httpx.Response(200, json=CATEGORIES))
    expenses = respx.get(f"{API}/expenses").mock(
        return_value=httpx.Response(200, json={"data": [], "total": 0, "page": 1, "page_size": 100})
    )

    result = await tools["spending_by_category"].ainvoke({"month": "2026-09", "category": "transporte"})

    params = expenses.calls[0].request.url.params
    assert (params["category_id"], params["start_date"], params["end_date"]) == (
        "c-transport",
        "2026-09-01",
        "2026-09-30",
    )
    assert result["category"] == "transporte" and result["count"] == 0


@respx.mock
async def test_monthly_totals_can_be_asked_for_a_given_month(tools) -> None:
    respx.get(f"{API}/reports/monthly").mock(
        return_value=httpx.Response(
            200,
            json=[{"month": f"2026-0{m}", "total_spent": 100.0, "total_earned": 300.0} for m in range(1, 10)],
        )
    )

    result = await tools["monthly_totals"].ainvoke({"month": "2026-07", "last_n_months": 1})

    assert [m["month"] for m in result["months"]] == ["2026-07"]
