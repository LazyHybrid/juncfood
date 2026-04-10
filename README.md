# Chore Clash

Chore Clash is a small hackathon game for settling household chores with quick, random mini-challenges.

Players are entered in a roster, chores are listed in prize order, and the app draws a random challenge round. After each matchup, every player casts a vote and the app assigns chores based on the final ranking.

Rankings are based only on recorded votes. If multiple players finish with the same vote total, their order is randomized before chores are assigned.

## Stack

- Go standard library API in `backend`
- React + TypeScript + Vite frontend in `frontend`

## Editable prompt data

- Mime scenarios live in `backend/data/mime_scenarios.txt`
- Tongue twisters live in `backend/data/tongue_twisters.txt`
- Trivia questions live in `backend/data/trivia_questions.txt`

Each file uses one prompt per line. Blank lines and lines starting with `#` are ignored.

## Single executable build

You can package the app as a single Go binary that serves both the API and the built React frontend.

If you download the project for Windows, a ready-to-run executable is already included at `chore-clash.exe` in the project root.

Run the included Windows executable:

```bash
./chore-clash.exe
```

The bundled app runs on `http://localhost:8080`.

Build for the current machine:

```bash
make build-app
```

That outputs:

- `release/chore-clash` on Linux/macOS

Build a Windows executable:

```bash
make build-exe
```

That outputs:

- `release/chore-clash.exe`

The repository also keeps a copy at:

- `chore-clash.exe`

When you run the bundled binary, it serves the full app from one process on `http://localhost:8080`.

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
make build-app
make build-exe
```

## API summary

- `GET /api/health` returns a basic health response.
- `GET /api/challenges` returns the built-in challenge deck.
- `POST /api/rounds` accepts players and chores, then returns a random challenge and seeded matchups.
- `POST /api/assignments` accepts vote results plus the original seed order, then returns ranked chore assignments.

## Game flow

1. Add one player per line.
2. Add chores from best outcome to worst outcome.
3. Toggle any challenges on or off in the challenge deck.
4. Draw a challenge round.
5. Run each matchup in real life.
6. Have every player vote on the active matchup.
7. Lock the assignments and use the result as the chore draft.

## Verification

- Backend tests: `cd backend && go test ./...`
- Frontend build: `cd frontend && npm run build`
- Bundled app build: `make build-app`
- Windows executable build: `make build-exe`