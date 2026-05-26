.PHONY: lint test cover build govulncheck

lint:
	golangci-lint run ./...

test:
	go test -race -coverprofile=coverage.out ./...

cover: test
	go tool cover -html=coverage.out

build:
	go build ./...

govulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

ci: lint test build govulncheck
