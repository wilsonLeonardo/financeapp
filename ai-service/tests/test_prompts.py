from datetime import date

import pytest

from app.agents.prompts import analyst, reply_language


@pytest.mark.parametrize(
    ("text", "language"),
    [
        ("Oi! O que você consegue fazer?", "Portuguese"),
        ("Quanto gastei com Uber em setembro?", "Portuguese"),
        ("Tenho transações sem categoria?", "Portuguese"),
        ("How much did I spend on groceries?", "English"),
        ("hi there", "English"),
    ],
)
def test_reply_language(text: str, language: str) -> None:
    assert reply_language(text) == language


def test_prompts_state_the_language_and_currency() -> None:
    prompt = analyst(date(2026, 10, 6), "Portuguese")
    assert "Always reply in Portuguese" in prompt and "like $1,234.56" in prompt
