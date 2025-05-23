.PHONY: setup test lint

setup:
	@which golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

test:
	go test -count=1 ./... -covermode=atomic -race

lint:
	golangci-lint run -c golangci.yml ./...