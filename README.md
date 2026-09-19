# FinanceApp

Personal finance tracker: log income and expenses, organise them into categories,
import bank statements, and see where the money went.

## Stack

| Layer    | Technology                                                       |
| -------- | ---------------------------------------------------------------- |
| Backend  | Go 1.25, Gin, GORM, PostgreSQL 16, Redis 7, JWT                  |
| Frontend | React 18, TypeScript, Vite, Tailwind CSS, TanStack Query, Recharts |
| Infra    | Docker Compose, Nginx (frontend image)                            |

## Features

- Email and password accounts with JWT sessions, revocable through Redis
- Income and expense tracking with categories, tags and free-text descriptions
- Filtering by date range, transaction type and category — including transactions
  with no category at all
- Monthly summaries and per-category breakdowns, charted on the dashboard
- Bank statement import from CSV and OFX/QFX, with a revert for a bad import

## Layout

```
backend/
├── cmd/server/          # entrypoint: config, connections, wiring
├── docs/                # generated OpenAPI spec (make docs)
├── internal/
│   ├── auth/            # one package per domain, each with
│   ├── category/        #   handler.go   HTTP in, JSON out
│   ├── expense/         #   service.go   business rules
│   ├── importer/        #   repository.go persistence
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
```

Handlers never touch the database and never know their own URL; services hold
the rules; repositories hold the SQL. Mocks are generated from the Repository
and Service interfaces with `make generate`.

## Running it

Everything runs from Docker Compose:

```bash
cp .env.example .env   # adjust the secrets before exposing anything publicly
make up                # or: docker compose up --build
```

`make help` lists every target.

| Service  | URL                            |
| -------- | ------------------------------ |
| Frontend | http://localhost:3000          |
| API      | http://localhost:8080/api/v1   |
| Health   | http://localhost:8080/health   |

### Running the parts on their own

```bash
# API — needs PostgreSQL and Redis reachable (docker compose up postgres redis)
cd backend && go run ./cmd/server

# Web app
cd frontend && npm install && npm run dev
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

The defaults exist so the stack boots with no setup. Set a real `JWT_SECRET` and
real database credentials before running this anywhere but your own machine.

## API

All routes live under `/api/v1`, grouped as `/auth`, `/expenses`, `/categories`,
`/reports` and `/imports`. Everything except register and login needs an
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

## Tests

```bash
make test        # Go unit tests
make web-test    # Vitest + Testing Library
make check       # everything CI runs
make cover       # Go coverage report
```

Services and handlers are covered with mocks generated by mockgen, and every
package under `pkg/` has its own tests.

CI runs both suites, `go vet`, a `gofmt` check and a production build of the
frontend on every push and pull request.
