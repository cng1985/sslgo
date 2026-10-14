GO ?= go
BIN ?= bin/sslplatform

.PHONY: web test build run

web:
	cd web && npm install && npm run build

test:
	$(GO) test ./...

build: web
	CGO_ENABLED=0 $(GO) build -o $(BIN) ./cmd/server

run: build
	./$(BIN)
