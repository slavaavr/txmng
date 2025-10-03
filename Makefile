.PHONY: setup test lint generate

setup:
	@which golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@which mockgen || go install go.uber.org/mock/mockgen@latest

test:
	go test -count=1 ./... -covermode=atomic -race

lint: setup
	golangci-lint run -c golangci.yml ./...

generate: setup
	go generate ./...