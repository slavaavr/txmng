.PHONY: test lint

test:
	go test -count=1 ./... -covermode=atomic -race

lint:
	golangci-lint run -c golangci.yml ./...