# Chore-Ha

Chore-Ha is a small hackathon game for settling household chores with quick, random mini-challenges.

Players are entered in a roster, chores are listed in prize order, and the app draws a random challenge round. After each matchup, every player casts a vote and the app assigns chores based on the final ranking.

Rankings are based only on recorded votes. If multiple players finish with the same vote total, their order is randomized before chores are assigned.

## Quick start with the exe

You can package the app as a single Go binary that serves both the API and the built React frontend.

If you download the project for Windows, a ready-to-run executable is already included at `chore-ha.exe` in the project root.

Run the included Windows executable:

```bash
./chore-ha.exe
```

The bundled app opens `http://127.0.0.1:8080` only while the executable is running. Closing the exe closes the server and releases the port.

If you want to expose it on your network intentionally, run it with a different host binding:

```bash
HOST=0.0.0.0 ./chore-ha.exe
```

Build for the current machine:

```bash
make build-app
```

That outputs:

- `release/chore-ha` on Linux/macOS

Build a Windows executable:

```bash
make build-exe
```

That outputs:

- `release/chore-ha.exe`

The repository also keeps a copy at:

- `chore-ha.exe`

When you run the bundled binary, it serves the full app from one process on `http://127.0.0.1:8080` by default.

## Game flow

1. Add one player per line.
2. Add chores from best outcome to worst outcome.
3. Toggle any challenges on or off in the challenge deck.
4. Draw a challenge round.
5. Run each matchup in real life.
6. Have every player vote on the active matchup.
7. Lock the assignments and use the result as the chore draft.

## Stack

- Go standard library API in `backend`
- React + TypeScript + Vite frontend in `frontend`

## Editable prompt data

- Mime scenarios live in `backend/data/mime_scenarios.txt`
- Tongue twisters live in `backend/data/tongue_twisters.txt`
- Trivia questions live in `backend/data/trivia_questions.txt`

Each file uses one prompt per line. Blank lines and lines starting with `#` are ignored.

## If the app crashes and the port stays busy

Normally the port closes automatically when the program exits. If Windows still shows the port as busy, you can close it with built-in commands.

Command Prompt, direct kill for the app port:

```bat
for /f "tokens=5" %a in ('netstat -ano ^| findstr :8080') do taskkill /PID %a /F
```

Command Prompt, direct kill for the Vite dev port:

```bat
for /f "tokens=5" %a in ('netstat -ano ^| findstr :5173') do taskkill /PID %a /F
```

PowerShell, direct kill for the app port:

```powershell
Stop-Process -Id (Get-NetTCPConnection -LocalPort 8080).OwningProcess -Force
```

PowerShell, direct kill for the Vite dev port:

```powershell
Stop-Process -Id (Get-NetTCPConnection -LocalPort 5173).OwningProcess -Force
```

## Local development

### 1. Start the Go API

```bash
cd backend
go run .
```

The API runs on `http://127.0.0.1:8080` by default.

### 2. Start the React app

```bash
cd frontend
npm install
npm run dev
```

The Vite dev server runs on `http://127.0.0.1:5173` and proxies `/api` to the Go API while that dev process is running.

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

## Verification

- Backend tests: `cd backend && go test ./...`
- Frontend build: `cd frontend && npm run build`
- Bundled app build: `make build-app`
- Windows executable build: `make build-exe`