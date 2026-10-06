import logging
import time

from fastapi import HTTPException, status
from redis.asyncio import Redis
from redis.exceptions import RedisError

log = logging.getLogger(__name__)


async def enforce_rate_limit(redis: Redis, user_id: str, limit: int, window_seconds: int = 60) -> None:
    """Fixed-window limit per user. Every request runs a local model, so a few users could otherwise
    saturate the CPU for everyone."""
    key = f"ai:ratelimit:{user_id}:{int(time.time() // window_seconds)}"
    try:
        count = await redis.incr(key)
        if count == 1:
            await redis.expire(key, window_seconds)
    except RedisError:
        log.warning("redis unavailable, skipping rate limit")
        return
    if count > limit:
        raise HTTPException(status.HTTP_429_TOO_MANY_REQUESTS, "rate limit exceeded, try again in a minute")
