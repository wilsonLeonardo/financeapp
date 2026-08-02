# FinanceApp

Personal finance tracker: log income and expenses, organise them into categories,
import bank statements, and see where the money went.

## Stack

| Layer    | Technology                                                       |
| -------- | ---------------------------------------------------------------- |
| Backend  | Go 1.22, Gin, GORM, PostgreSQL 16, Redis 7, JWT                  |
| Frontend | React 18, TypeScript, Vite, Tailwind CSS, TanStack Query, Recharts |
| Infra    | Docker Compose, Nginx (frontend image)                            |

## Features

- Email and password accounts with JWT sessions, revocable through Redis
- Income and expense tracking with categories, tags and free-text descriptions
- Filtering by date range, transaction type and category — including transactions
  with no category at all
- Monthly summaries and per-category breakdowns, charted on the dashboard
- Bank statement import from CSV and OFX/QFX, with a revert for a bad import

## Running it

Everything runs from Docker Compose:

```bash
cp .env.example .env   # adjust the secrets before exposing anything publicly
docker compose up --build
```

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

All routes are under `/api/v1`. Everything except register and login requires an
`Authorization: Bearer <token>` header.

| Method   | Route                      | Description                             |
| -------- | -------------------------- | --------------------------------------- |
| `POST`   | `/auth/register`           | Create an account                       |
| `POST`   | `/auth/login`              | Exchange credentials for a token        |
| `POST`   | `/auth/logout`             | Revoke the current token                |
| `GET`    | `/expenses`                | List transactions (filters below)       |
| `POST`   | `/expenses`                | Create a transaction                    |
| `GET`    | `/expenses/:id`            | Fetch one transaction                   |
| `PUT`    | `/expenses/:id`            | Update a transaction                    |
| `DELETE` | `/expenses/:id`            | Delete a transaction                    |
| `GET`    | `/categories`              | List categories                         |
| `POST`   | `/categories`              | Create a category                       |
| `PUT`    | `/categories/:id`          | Update a category                       |
| `DELETE` | `/categories/:id`          | Delete a category                       |
| `GET`    | `/reports/monthly`         | Totals per month (`?months=12`)         |
| `GET`    | `/reports/categories`      | Totals per category for a date range    |
| `POST`   | `/imports`                 | Upload a statement (multipart `file`)   |
| `GET`    | `/imports`                 | List past imports                       |
| `DELETE` | `/imports/:id`             | Revert an import and its transactions   |

`GET /expenses` accepts `start_date`, `end_date` (both `YYYY-MM-DD`), `type`
(`expense` or `income`), `page`, `page_size`, and `category_id`. Passing
`category_id=none` returns only the transactions that have no category.

## Tests

```bash
cd backend  && go test ./...   # Go unit tests
cd frontend && npm test        # Vitest + Testing Library
```

CI runs both suites, `go vet`, a `gofmt` check and a production build of the
frontend on every push and pull request.
