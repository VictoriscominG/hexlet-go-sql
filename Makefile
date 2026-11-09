.PHONY: sqlc build
sqlc:
	go tool sqlc generate

build:
	go build ./...
