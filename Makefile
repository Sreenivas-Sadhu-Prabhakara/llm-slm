.PHONY: db-up db-down migrate test ask ingest
# This machine uses Colima + standalone docker-compose (no `docker compose` plugin).
db-up:        ; colima start && docker-compose up -d
db-down:      ; docker-compose down
migrate:      ; go run ./cmd/server -migrate
test:         ; go test ./...
ask:          ; go run ./cmd/ask
ingest:       ; go run ./cmd/ingest
