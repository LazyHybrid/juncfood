.PHONY: dev-backend dev-frontend test build sync-web build-app build-exe

dev-backend:
	cd backend && go run .

dev-frontend:
	cd frontend && npm run dev

test:
	cd backend && go test ./...

build:
	cd frontend && npm run build

sync-web:
	cd frontend && npm run build
	rm -rf backend/web
	mkdir -p backend/web
	cp -R frontend/dist/. backend/web/

build-app: sync-web
	cd backend && go build -o ../release/chore-ha .

build-exe: sync-web
	cd backend && GOOS=windows GOARCH=amd64 go build -o ../release/chore-ha.exe .