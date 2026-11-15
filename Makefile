BINARY  := ffcc-gci
VERSION ?= dev
LDFLAGS := -X ffcc-gci/internal/cli.Version=$(VERSION)

.PHONY: build test vet fmt check clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/ffcc-gci

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

# Everything that must pass before a change is considered done.
check: vet test
	@test -z "$$(gofmt -l .)" || { echo "gofmt needed on:"; gofmt -l .; exit 1; }

clean:
	rm -rf bin
