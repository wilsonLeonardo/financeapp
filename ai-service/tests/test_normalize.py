import pytest

from app.rag.normalize import index_text, normalize_description


@pytest.mark.parametrize(
    ("raw", "expected"),
    [
        ("COMPRA CARTAO 4321 IFOOD *RESTAURANTE 12/09", "ifood restaurante"),
        ("PAG*DROGASIL 0332", "drogasil"),
        ("NETFLIX.COM", "netflix"),
        ("PADARIA SÃO JOÃO LTDA", "padaria sao joao"),
        ("PIX ENVIADO MARIA SOUZA", "pix enviado maria souza"),
        ("UBER *TRIP 15/09/2026", "uber trip"),
    ],
)
def test_strips_bank_noise_but_keeps_the_merchant(raw: str, expected: str) -> None:
    assert normalize_description(raw) == expected


def test_same_merchant_with_different_noise_normalizes_identically() -> None:
    assert normalize_description("COMPRA CARTAO 1111 IFOOD *REST 01/09") == normalize_description(
        "IFOOD*REST 9988 23/10"
    )


def test_falls_back_to_the_lowercased_text_when_everything_is_noise() -> None:
    assert normalize_description("12/09 4321") == "12/09 4321"


def test_index_text_leaves_the_type_to_the_search_filter() -> None:
    assert index_text("Uber *Trip") == "uber trip"
