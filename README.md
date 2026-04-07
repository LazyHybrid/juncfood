# Chore Clash

Chore Clash is a small hackathon game for settling household chores with quick, random mini-challenges.

Players are entered in a roster, chores are listed in prize order, and the app draws a random challenge round. After each matchup, you pick the winner and the app assigns chores based on the final ranking.

Rankings are based only on recorded round wins. If multiple players finish with the same vote total, their order is randomized before chores are assigned.

## Stack

- Go standard library API in `backend`
- React + TypeScript + Vite frontend in `frontend`

## Editable prompt data

- Mime scenarios live in `backend/data/mime_scenarios.txt`
- Tongue twisters live in `backend/data/tongue_twisters.txt`

Each file uses one prompt per line. Blank lines and lines starting with `#` are ignored.

## Local development

### 1. Start the Go API

```bash
cd backend
go run .
```

The API runs on `http://localhost:8080`.

### 2. Start the React app

```bash
cd frontend
npm install
npm run dev
```

The Vite dev server runs on `http://localhost:5173` and proxies `/api` to the Go API.

## Quick commands

From the repo root:

```bash
make dev-backend
make dev-frontend
make test
make build
```

## API summary

- `GET /api/health` returns a basic health response.
- `GET /api/challenges` returns the built-in challenge deck.
- `POST /api/rounds` accepts players and chores, then returns a random challenge and seeded matchups.
- `POST /api/assignments` accepts winners plus the original seed order, then returns ranked chore assignments.

## Game flow

1. Add one player per line.
2. Add chores from best outcome to worst outcome.
3. Draw a challenge round.
4. Run each matchup in real life and click the winner.
5. Lock the assignments and use the result as the chore draft.

## Verification

- Backend tests: `cd backend && go test ./...`
- Frontend build: `cd frontend && npm run build`