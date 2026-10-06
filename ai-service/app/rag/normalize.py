"""Cleans bank statement descriptions before they are embedded.

Raw lines like "COMPRA CARTAO 4321 IFOOD *RESTAURANTE 12/09" vary in card digits, dates and
boilerplate that say nothing about the category. Stripping that noise makes the same merchant land
close to itself in embedding space, which is where most of the categorization accuracy comes from.
"""

import re
import unicodedata

_DATE = re.compile(r"\b\d{1,2}[/.-]\d{1,2}(?:[/.-]\d{2,4})?\b")
_DIGITS = re.compile(r"\d+")
_NON_LETTERS = re.compile(r"[^a-z\s]")

# Payment boilerplate and legal suffixes. Words such as "pix", "ted" or "boleto" stay: they hint at
# transfers and bills.
_STOPWORDS = frozenset(
    {
        "compra", "cartao", "card", "debito", "credito", "pag", "pagto", "pgto", "pagamento",
        "parcela", "parc", "com", "www", "br", "ltda", "sa", "me", "eireli", "de", "da", "do", "em",
    }
)  # fmt: skip


def normalize_description(text: str) -> str:
    ascii_text = unicodedata.normalize("NFKD", text).encode("ascii", "ignore").decode().lower()
    cleaned = _NON_LETTERS.sub(" ", _DIGITS.sub(" ", _DATE.sub(" ", ascii_text)))
    tokens = [t for t in cleaned.split() if len(t) > 1 and t not in _STOPWORDS]
    return " ".join(tokens) or ascii_text.strip()


def index_text(description: str) -> str:
    """The text embedded for one transaction.

    The type stays out of it on purpose: it is already a search filter, and a word shared by every
    transaction inflates all similarities alike, letting unrelated merchants clear the kNN threshold.
    """
    return normalize_description(description)
