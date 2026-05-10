.PHONY: dev build test clean

dev:
	cd backend && go run cmd/server/main.go &
	cd frontend && npm run dev

build:
	cd backend && CGO_ENABLED=0 go build -o agent-os ./cmd/server
	cd frontend && npm run build

test:
	cd backend && go test ./... -v
	cd backend && go test ./bench -bench=. -benchmem

clean:
	rm -f backend/agent-os
	rm -rf frontend/dist
	rm -rf state/ telemetry/
