"""Generates the synthetic labeled dataset used by the evals and the demo seed.

Four months of Brazilian bank-statement lines, written the messy way banks write them: card numbers,
dates, "PAG*" and "COMPRA CARTAO" prefixes, city suffixes, inconsistent casing. Recurring merchants
(rent, subscriptions, the usual supermarket) repeat across months; some merchants only show up in the
last month, so the eval also measures transactions the history has never seen.

Everything is fictional and seeded, so the file is reproducible: python -m evals.generate_dataset
"""

import json
import random
from datetime import date, timedelta
from pathlib import Path

OUT = Path(__file__).parent / "data" / "transactions.jsonl"
MONTHS = [(2026, 6), (2026, 7), (2026, 8), (2026, 9)]
SEED = 42

# category -> (merchants, amount range, transactions per month)
EXPENSES: dict[str, tuple[list[str], tuple[float, float], int]] = {
    "Alimentação": (
        ["IFOOD *RESTAURANTE", "IFOOD *LANCHONETE", "RAPPI*RESTAURANTE", "PADARIA DOCE PAO", "OUTBACK STEAKHOUSE",
         "MC DONALDS", "BURGER KING", "RESTAURANTE SABOR CASEIRO", "STARBUCKS", "SUSHI YAMA"],
        (18, 160), 12,
    ),
    "Mercado": (
        ["CARREFOUR HIPER", "PAO DE ACUCAR", "ASSAI ATACADISTA", "EXTRA SUPERMERCADO", "SONDA SUPERMERCADOS",
         "HORTIFRUTI QUITANDA", "ST MARCHE"],
        (35, 480), 7,
    ),
    "Transporte": (
        ["UBER *TRIP", "99 *POP", "POSTO SHELL", "POSTO IPIRANGA", "SEM PARAR", "ESTAPAR ESTACIONAMENTO",
         "METRO SP RECARGA", "AUTO POSTO BR"],
        (9, 260), 10,
    ),
    "Moradia": (
        ["ALUGUEL IMOBILIARIA LAR", "CONDOMINIO ED SOLAR", "ENEL SP ENERGIA", "SABESP AGUA", "VIVO FIBRA INTERNET",
         "COMGAS"],
        (60, 2400), 5,
    ),
    "Saúde": (
        ["DROGASIL", "DROGA RAIA", "PAGUE MENOS FARMACIA", "UNIMED MENSALIDADE", "SMART FIT", "LAB FLEURY",
         "CLINICA ODONTO SORRIA"],
        (25, 650), 4,
    ),
    "Lazer": (
        ["CINEMARK", "INGRESSO.COM", "SYMPLA EVENTOS", "BAR DO ZE", "STEAM GAMES", "LIVRARIA CULTURA CAFE",
         "PARQUE IBIRAPUERA ESTAC"],
        (20, 320), 4,
    ),
    "Assinaturas": (
        ["NETFLIX.COM", "SPOTIFY", "AMAZON PRIME", "YOUTUBE PREMIUM", "APPLE.COM/BILL ICLOUD", "DISNEY PLUS",
         "GLOBOPLAY"],
        (10, 60), 4,
    ),
    "Compras": (
        ["AMAZON MARKETPLACE", "MERCADOLIVRE*VENDEDOR", "SHOPEE", "MAGALU", "RENNER", "KABUM", "LEROY MERLIN",
         "C&A MODA"],
        (40, 900), 4,
    ),
    "Educação": (
        ["ALURA CURSOS", "UDEMY", "COURSERA", "ESCOLA DE INGLES WISE UP", "AMAZON KINDLE EBOOK"],
        (30, 420), 2,
    ),
    "Transferências": (
        ["PIX ENVIADO MARIA SOUZA", "PIX ENVIADO JOAO PEREIRA", "TED ENVIADA ANA LIMA", "PIX ENVIADO CARLOS ROCHA"],
        (50, 800), 2,
    ),
}  # fmt: skip

INCOME: dict[str, tuple[list[str], tuple[float, float], int]] = {
    "Renda": (
        ["SALARIO EMPRESA TECH LTDA", "PIX RECEBIDO FREELANCE", "RENDIMENTO CDB", "REEMBOLSO DESPESAS EMPRESA"],
        (300, 14000), 3,
    ),
}  # fmt: skip

# Merchants that only appear in the last month: the history cannot have seen them.
NEW_IN_LAST_MONTH = {
    "Alimentação": ["ZE DELIVERY BEBIDAS", "CANTINA DO NONNO"],
    "Mercado": ["OXXO MINIMERCADO", "DIA SUPERMERCADO"],
    "Transporte": ["INDRIVE CORRIDA", "ZONA AZUL DIGITAL"],
    "Saúde": ["DROGARIA SAO PAULO", "TOTALPASS"],
    "Lazer": ["TICKETMASTER BRASIL", "BOLICHE STRIKE"],
    "Assinaturas": ["MAX STREAMING", "DEEZER PREMIUM"],
    "Compras": ["ALIEXPRESS", "CENTAURO ESPORTES"],
    "Educação": ["DESCOMPLICA CURSO"],
}

_PREFIXES = ["", "", "COMPRA CARTAO {card} ", "PAG*", "DEB AUT ", "COMPRA "]
_SUFFIXES = ["", "", " SAO PAULO BR", " {dd}/{mm}", " *{code}", " {card}"]


def _noisy(merchant: str, day: date, rng: random.Random) -> str:
    card = f"{rng.randint(1000, 9999)}"
    code = "".join(rng.choices("ABCDEFGHJKLMNPQRSTUVWXYZ0123456789", k=6))
    text = rng.choice(_PREFIXES) + merchant + rng.choice(_SUFFIXES)
    text = text.format(card=card, dd=f"{day.day:02d}", mm=f"{day.month:02d}", code=code)
    return text.lower().title() if rng.random() < 0.15 else text


def _days(year: int, month: int, rng: random.Random, n: int) -> list[date]:
    first = date(year, month, 1)
    last = (first.replace(day=28) + timedelta(days=4)).replace(day=1) - timedelta(days=1)
    return sorted(first + timedelta(days=rng.randint(0, (last - first).days)) for _ in range(n))


def generate() -> list[dict]:
    rng = random.Random(SEED)
    rows: list[dict] = []
    for idx, (year, month) in enumerate(MONTHS):
        last_month = idx == len(MONTHS) - 1
        for txn_type, table in (("expense", EXPENSES), ("income", INCOME)):
            for category, (merchants, (low, high), per_month) in table.items():
                pool = merchants + (NEW_IN_LAST_MONTH.get(category, []) * 2 if last_month else [])
                for day in _days(year, month, rng, per_month):
                    merchant = rng.choice(pool)
                    rows.append(
                        {
                            "date": day.isoformat(),
                            "description": _noisy(merchant, day, rng),
                            "amount": round(rng.uniform(low, high), 2),
                            "type": txn_type,
                            "category": category,
                            "merchant": merchant,
                        }
                    )
    rows.sort(key=lambda r: r["date"])
    for i, row in enumerate(rows, 1):
        row["id"] = f"t{i:04d}"
    return rows


def main() -> None:
    rows = generate()
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text("".join(json.dumps(r, ensure_ascii=False) + "\n" for r in rows))
    print(f"wrote {len(rows)} transactions to {OUT}")


if __name__ == "__main__":
    main()
