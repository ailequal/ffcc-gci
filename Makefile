BINARY  := ffcc-gci

# The version stamped into the program: the git tag when the commit has one,
# otherwise the tag and revision, or "dev" outside a git checkout. The
# release workflow passes VERSION itself.
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/ailequal/ffcc-gci/internal/cli.Version=$(VERSION)

# golangci-lint release used by lint-docker and by the workflow.
GOLANGCI_LINT_VERSION := v2.14.0

.PHONY: build install dist test vet fmt lint lint-docker docs check clean

# Build the program into bin/.
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/ffcc-gci

# Install the program into Go's bin directory, so it runs as "ffcc-gci".
install:
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/ffcc-gci

# Cross-compile the release archives for every supported platform into dist/.
dist:
	sh scripts/dist.sh $(VERSION)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -s -l -w .

# Needs golangci-lint (https://golangci-lint.run).
lint:
	golangci-lint run ./...

# Run the pinned golangci-lint in Docker, with no local install.
lint-docker:
	docker run --rm -v "$(CURDIR)":/src:ro -w /src -e GOFLAGS=-buildvcs=false \
		golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run ./...

# Regenerate docs/USAGE.md from the program's own help.
docs:
	go run ./cmd/ffcc-gci help -markdown > docs/USAGE.md

# Everything that must pass before a change is considered done. The tests
# also fail when docs/USAGE.md is out of date.
check: vet test
	@test -z "$$(gofmt -s -l .)" || { echo "gofmt needed on:"; gofmt -s -l .; exit 1; }
	sh scripts/changelog.sh check

clean:
	rm -rf bin
