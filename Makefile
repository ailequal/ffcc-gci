BINARY  := ffcc-gci
VERSION ?= dev
LDFLAGS := -X github.com/ailequal/ffcc-gci/internal/cli.Version=$(VERSION)

.PHONY: build install test vet fmt lint docs check clean

# Build the program into bin/.
build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/ffcc-gci

# Install the program into Go's bin directory, so it runs as "ffcc-gci".
install:
	go install -ldflags "$(LDFLAGS)" ./cmd/ffcc-gci

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -s -l -w .

# Needs golangci-lint (https://golangci-lint.run).
lint:
	golangci-lint run ./...

# Regenerate docs/USAGE.md from the program's own help.
docs:
	go run ./cmd/ffcc-gci help -markdown > docs/USAGE.md

# Everything that must pass before a change is considered done. The tests
# also fail when docs/USAGE.md is out of date.
check: vet test
	@test -z "$$(gofmt -s -l .)" || { echo "gofmt needed on:"; gofmt -s -l .; exit 1; }

clean:
	rm -rf bin
