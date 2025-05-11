.PHONY: test
test:
	go test -count 1 ./...

.PHONY: fuzz
fuzz:
	go test -fuzz . -fuzztime=1m ./...

.PHONY: lint
lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...
