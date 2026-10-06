"""Authenticates requests with the JWTs the Go API issues.

The ai-service needs a user ID it can trust, because every vector search is scoped by it, so it
verifies tokens itself with the shared secret instead of trusting a header. Logout is honoured the
same way the Go middleware does it: a revoked token lives in Redis under ``blacklist:<token>``.
"""

import logging
from dataclasses import dataclass

import jwt
from fastapi import HTTPException, status
from redis.asyncio import Redis
from redis.exceptions import RedisError

log = logging.getLogger(__name__)


@dataclass(frozen=True)
class Principal:
    user_id: str
    token: str


def _unauthorized(message: str) -> HTTPException:
    return HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail=message)


async def authenticate(authorization: str | None, secret: str, redis: Redis) -> Principal:
    if not authorization:
        raise _unauthorized("authorization header required")

    scheme, _, token = authorization.partition(" ")
    if scheme.lower() != "bearer" or not token:
        raise _unauthorized("invalid authorization format")

    # Like the Go middleware, a Redis outage does not lock everyone out: the token is still
    # verified below, only revocation is skipped.
    try:
        if await redis.exists(f"blacklist:{token}"):
            raise _unauthorized("token revoked")
    except RedisError:
        log.warning("redis unavailable, skipping token revocation check")

    try:
        claims = jwt.decode(token, secret, algorithms=["HS256"])
    except jwt.PyJWTError as exc:
        raise _unauthorized("invalid or expired token") from exc

    user_id = claims.get("user_id")
    if not user_id:
        raise _unauthorized("invalid or expired token")
    return Principal(user_id=str(user_id), token=token)
