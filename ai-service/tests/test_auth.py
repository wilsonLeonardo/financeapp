import pytest
from fastapi import HTTPException

from app.auth import authenticate
from tests.conftest import SECRET, FakeRedis, make_token


async def test_valid_token_yields_the_user() -> None:
    token = make_token("abc")
    principal = await authenticate(f"Bearer {token}", SECRET, FakeRedis())
    assert principal.user_id == "abc"
    assert principal.token == token


@pytest.mark.parametrize(
    "header",
    [
        None,
        "",
        "Basic xyz",
        "Bearer",
        f"Bearer {make_token(secret='other')}",
        f"Bearer {make_token(expires_in=-10)}",
    ],
)
async def test_rejects_missing_malformed_forged_and_expired_tokens(header: str | None) -> None:
    with pytest.raises(HTTPException) as exc:
        await authenticate(header, SECRET, FakeRedis())
    assert exc.value.status_code == 401


async def test_honours_logout_blacklist() -> None:
    token = make_token()
    with pytest.raises(HTTPException) as exc:
        await authenticate(f"Bearer {token}", SECRET, FakeRedis(blacklist={token}))
    assert exc.value.detail == "token revoked"


async def test_redis_outage_still_verifies_the_token() -> None:
    principal = await authenticate(f"Bearer {make_token('abc')}", SECRET, FakeRedis(fail=True))
    assert principal.user_id == "abc"
