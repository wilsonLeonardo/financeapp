# FinanceApp

Personal finance tracker: log income and expenses, organise them into categories,
see where the money went, and ask a local AI assistant about it.

## Stack

| Layer    | Technology                                                       |
| -------- | ---------------------------------------------------------------- |
| Backend  | Go 1.25, Gin, GORM, PostgreSQL 16 (pgvector), Redis 7, JWT       |
| AI       | Python 3.12, FastAPI, LangChain, LangGraph, Ollama (local models) |
| Frontend | React 18, TypeScript, Vite, Tailwind CSS, TanStack Query, Recharts, Vercel AI SDK |
| Infra    | Docker Compose, Nginx (frontend image)                            |

## Features

- Email and password accounts with JWT sessions, revocable through Redis
- Income and expense tracking with categories, tags and free-text descriptions
- Filtering by date range, transaction type and category — including transactions
  with no category at all
- Monthly summaries and per-category breakdowns, charted on the dashboard
- An AI assistant that answers questions about your money through tool calls to the API
- Category suggestions for uncategorized transactions, learned from your own history
- Everything AI runs on local models: no transaction ever leaves the machine

## Layout

```
backend/
├── cmd/server/          # entrypoint: config, connections, wiring
├── docs/                # generated OpenAPI spec (make docs)
├── internal/
│   ├── auth/            # one package per domain, each with
│   ├── category/        #   handler.go    HTTP in, JSON out
│   ├── expense/         #   service.go    business rules
│   │                    #   repository.go persistence
│   ├── routes/          # the whole URL table in one place
│   ├── domain/          # entities shared across domains
│   ├── database/        # postgres and redis connections
│   └── testutils/       # test helpers and generated mocks
└── pkg/                 # reusable, no business logic
    ├── config/          # environment loading
    ├── errors/          # AppError, carrying an HTTP status
    ├── logger/          # slog setup: text in dev, JSON in production
    ├── middleware/      # auth and CORS gin middleware
    ├── response/        # one JSON error shape for every handler
    └── security/        # bcrypt hashing and JWT signing

ai-service/
├── app/
│   ├── agents/          # LangGraph router and specialists, their prompts and tools
│   ├── rag/             # description normalization, pgvector index, categorizer
│   ├── streaming.py     # LangGraph run -> AI SDK UI message stream
│   └── main.py          # FastAPI endpoints
├── evals/               # labeled datasets and the accuracy runners
├── scripts/             # demo account seeding
└── tests/               # pytest suite; no model, database or Redis needed
```

Handlers never touch the database and never know their own URL; services hold
the rules; repositories hold the SQL. Mocks are generated from the Repository
and Service interfaces with `make generate`.

## Running it

Everything runs from Docker Compose:

```bash
cp .env.example .env   # adjust the secrets before exposing anything publicly
make up                # or: docker compose up --build
make ai-install        # creates the ai-service virtualenv the seed and eval targets use
make ai-seed           # optional: a demo account with four months of transactions
```

`make help` lists every target.

| Service    | URL                            |
| ---------- | ------------------------------ |
| Frontend   | http://localhost:3000          |
| API        | http://localhost:8080/api/v1   |
| Health     | http://localhost:8080/health   |
| AI service | http://localhost:8000/health   |
| Ollama     | http://localhost:11434         |

The first `make up` also downloads the local models (~2.2 GB) before the AI
service starts, so it takes a few minutes; later starts skip the download and
work offline. The demo account is `demo@financeapp.dev` / `demo-password-123`. Every host port
can be changed in `.env` (`*_HOST_PORT`) when another stack already uses it.

### Running the parts on their own

```bash
# API — needs PostgreSQL and Redis reachable (docker compose up postgres redis)
cd backend && go run ./cmd/server

# Web app
cd frontend && npm install && npm run dev

# AI service — needs the API, PostgreSQL, Redis and Ollama reachable
make ai-install && make ai-dev
```

## Configuration

Every variable is read from the environment and falls back to a development
default. `.env.example` lists them all; `.env` is not tracked.

| Variable                               | Default                       |
| -------------------------------------- | ----------------------------- |
| `APP_ENV`                              | `development`                 |
| `LOG_LEVEL`                            | `info`                        |
| `SERVER_PORT`                          | `8080`                        |
| `POSTGRES_HOST` / `POSTGRES_PORT`      | `localhost` / `5432`          |
| `POSTGRES_USER` / `POSTGRES_PASSWORD`  | `financeapp` / `financeapp123`|
| `POSTGRES_DB`                          | `financeapp`                  |
| `REDIS_HOST` / `REDIS_PORT`            | `localhost` / `6379`          |
| `REDIS_PASSWORD`                       | `redis123`                    |
| `JWT_SECRET`                           | development placeholder       |
| `JWT_EXPIRY_HOURS`                     | `24`                          |
| `VITE_API_URL`                         | `http://localhost:8080/api/v1`|
| `VITE_AI_URL`                          | `http://localhost:8000`       |
| `CHAT_MODEL` / `EMBED_MODEL`           | `qwen2.5:3b` / `nomic-embed-text` |
| `*_HOST_PORT`                          | the ports in the table above  |

The defaults exist so the stack boots with no setup. Set a real `JWT_SECRET` and
real database credentials before running this anywhere but your own machine.

## API

All routes live under `/api/v1`, grouped as `/auth`, `/expenses`, `/categories`
and `/reports`. Everything except register and login needs an
`Authorization: Bearer <token>` header.

The full reference — every parameter, payload and response — is the OpenAPI
spec, served as an interactive UI while the stack is running:

**[localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)**

Click **Authorize** and paste `Bearer <token>` from `/auth/login` to call the
protected endpoints straight from the browser. Reading the repo rather than
running it? The same content is committed as
[`backend/docs/swagger.yaml`](backend/docs/swagger.yaml).

The spec is generated from the annotations above each handler, so it is only as
current as the last `make docs`, and it is not mounted when
`APP_ENV=production`.

## AI assistant

The `ai-service` adds two features on top of the API, both running on local
models through Ollama (`qwen2.5:3b` for chat, `nomic-embed-text` for embeddings).

```
React (useChat) ──SSE──▶ ai-service (FastAPI) ──JWT──▶ Go API ──▶ PostgreSQL
                              │                                      ▲
                              ├── LangGraph agents ──▶ Ollama        │
                              └── categorization index ──────────────┘ (pgvector)
```

**Assistant (Assistente page).** A LangGraph graph routes each message to one
specialist: an analyst with read-only finance tools, a categorizer, or a general
responder. The router is a single structured-output call, and each specialist sees
only the tools it needs, which keeps a 3B model's tool calls reliable. Tools call
the Go API with the user's own token and return totals already computed, so the
model never does arithmetic. Responses stream to the browser in the Vercel AI SDK
protocol, so the front end uses `useChat` with no custom parsing.

**Category suggestions (Transactions page).** For each uncategorized transaction,
the user's most similar categorized transactions are retrieved from pgvector. When
the closest ones are near-identical and agree, their category is used without
calling the model; otherwise the model picks one of the user's categories, with
those neighbours as examples and a JSON schema that only allows real category
names. Nothing is saved until the user confirms, and every confirmation feeds the
index.

Design choices worth knowing:

- **Per-user isolation.** Every vector search is filtered by the user ID taken from
  the verified JWT, never from the request body.
- **Read-only agents.** The assistant can suggest but not change data; writes only
  happen from the review table, after confirmation.
- **Graceful degradation.** If the vector index is down, chat keeps working and
  only categorization reports itself unavailable.
- **Bounded cost.** Each user is rate limited in Redis, and the tool loop has a
  step limit, because every request runs a model on the local CPU.

### Evals

`make ai-eval` replays a synthetic, seeded dataset (`ai-service/evals/data`):
four months of bank-statement-style transactions, of which the first three are
the indexed history (171 transactions) and the last one is the test set (57),
so retrieval never sees the answers. 21 of the test transactions come from
merchants the history has never seen.

Results on `qwen2.5:3b` + `nomic-embed-text`, CPU only:

| Strategy               | Accuracy | Seen merchants | New merchants | Macro F1 | No model call | p50   |
| ---------------------- | -------- | -------------- | ------------- | -------- | ------------- | ----- |
| Zero-shot (model only) | 45.6%    | 41.7%          | 52.4%         | 0.505    | 0%            | 1.8 s |
| kNN (history only)     | 64.9%    | 86.1%          | 28.6%         | 0.574    | 100%          | 36 ms |
| **RAG (both)**         | **89.5%**| **100%**       | **71.4%**     | **0.902**| **54%**       | 66 ms |

Retrieval handles recurring merchants without touching the model, and the
model, given the user's own examples per category, handles the new ones.
`make ai-eval-router` scores the router on 36 labeled prompts: 94.4% routed to
the right specialist. `make ai-eval-tools` scores the analyst's tool calls on 21
labeled questions, 8 of them follow-ups ("and the details of that category?"):
81% pick the right tool with the right arguments (75% of follow-ups), up from
60% (43%) before the analyst had to ground every turn in a tool call and the
tools learned to accept the arguments a small model reaches for first.
Rewriting follow-ups into standalone questions was also tried and dropped: the
3B model drifted while rewriting, and the score fell to 76% (62%). Full results, including every miss, are committed under
`ai-service/evals/results/`.

The dataset was also used while developing the pipeline, so read these numbers
as indicative rather than as a benchmark.

### Upgrading an existing database

The Postgres image moved from `postgres:16-alpine` to `pgvector/pgvector:pg16`,
which is Debian-based. Text sorts differently under glibc than under Alpine's
musl, so after switching an existing volume, rebuild the text indexes once:

```bash
make db   # then, in psql:
REINDEX DATABASE financeapp;
```

## Tests

```bash
make test        # Go unit tests
make web-test    # Vitest + Testing Library
make ai-test     # pytest for the ai-service
make check       # everything CI runs
make cover       # Go coverage report
make ai-eval     # categorization accuracy on the local models (not in CI)
```

Services and handlers are covered with mocks generated by mockgen, and every
package under `pkg/` has its own tests. The ai-service suite fakes the models,
Postgres and Redis, so it runs anywhere; the evals need Ollama and run locally.

CI runs the three suites, `go vet`, `gofmt` and `ruff` checks and a production
build of the frontend on every push and pull request.
