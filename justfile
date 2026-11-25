# ffcc-gci command runner: https://just.systems
#
# Run `just` to list the recipes by group. Useful to know:
#
#   just --show check           print a recipe without running it
#   just --dry-run dist         print the commands a recipe would run
#   just --usage test-terminal  print the arguments a recipe takes
#   just --choose               pick a recipe from a menu (needs fzf)
#   just version=v1.2.3 dist    set a variable for one run
#   just --evaluate             print the variables, such as the version
#   just --fmt                  format this file (`just fmt` does it too)
#   just --completions bash     print a shell completion script
#
# Recipes run from the directory of this file, whatever directory `just` is
# called from, and stop at the first command that fails. They are written for
# sh: on Windows, run `just` from Git Bash.

set minimum-version := "1.55.0"

# -e stops a recipe line at its first failing command, -u at an unset variable.

set shell := ["sh", "-eu", "-c"]

# The version stamped into the program: the git tag when the commit has one,
# otherwise the tag and revision, or "dev" outside a git checkout. The release
# workflow sets it itself: just version=vX.Y.Z dist

version := `git describe --tags --always 2>/dev/null || echo dev`
ldflags := "-s -w -X github.com/ailequal/ffcc-gci/internal/cli.Version=" + version

# golangci-lint release used by lint-docker and by the workflow.

golangci_lint_version := "v2.15.0"

# List the recipes
[private]
default:
    @{{ quote(just_executable()) }} --justfile {{ quote(justfile()) }} --list --unsorted

# Build the program into bin/ffcc-gci, stamped with the version from git
[group('build')]
build:
    go build -trimpath -ldflags {{ quote(ldflags) }} -o bin/ffcc-gci ./cmd/ffcc-gci

# Install the program into Go's bin directory, so that it runs as "ffcc-gci"
[group('build')]
install:
    go install -trimpath -ldflags {{ quote(ldflags) }} ./cmd/ffcc-gci

# Build the program and run it with ARGS, such as: just run menu save.gci
[group('build')]
[positional-arguments]
run *args: build
    exec bin/ffcc-gci "$@"

# The archives are one tar.gz per platform (zip on Windows) and a SHA256SUMS
# file; the release workflow publishes exactly these.

# Cross-compile the release archives for every platform into dist/
[group('build')]
dist:
    sh scripts/dist.sh {{ quote(version) }}

# Remove the build output: bin/ and dist/
[group('build')]
clean:
    rm -rf bin dist

# Run all that must pass before a commit: format, vet, tests, changelog
[group('check')]
check: fmt-check vet test changelog-check

# The terminal tests start the built program in a tmux session and type into
# it. They are the only tests that see the menu as a user does.

# Run the tests; the terminal tests need tmux and are skipped without it
[group('check')]
test:
    go test ./...

# Run the tests without the terminal tests
[group('check')]
test-short:
    go test -short ./...

# Run the tests with the race detector
[group('check')]
test-race:
    go test -race -count=1 ./...

# Run the tests and print the coverage of each function
[group('check')]
test-cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Repeating them is how a lost key press shows up: just test-terminal 30

# Run only the terminal tests, COUNT times; fails without tmux
[arg('count', pattern='[1-9][0-9]*', help='how many times to run them')]
[group('check')]
test-terminal count='1':
    FFCC_GCI_E2E=require go test -count={{ count }} -run InATerminal ./internal/cli

# Run go vet
[group('check')]
vet:
    go vet ./...

# Format the Go sources with gofmt, and this file with just
[group('check')]
fmt:
    gofmt -s -w .
    {{ quote(just_executable()) }} --justfile {{ quote(justfile()) }} --fmt

# Report formatting differences without writing changes
[group('check')]
fmt-check:
    @test -z "$(gofmt -s -l .)" || { echo "gofmt is needed on:"; gofmt -s -l .; exit 1; }
    {{ quote(just_executable()) }} --justfile {{ quote(justfile()) }} --fmt --check

# Run golangci-lint, which must be installed (https://golangci-lint.run)
[group('check')]
lint:
    golangci-lint run ./...

# Run the pinned golangci-lint in Docker, as the workflow does
[group('check')]
lint-docker:
    docker run --rm -v {{ quote(justfile_directory()) }}:/src:ro -w /src -e GOFLAGS=-buildvcs=false \
      golangci/golangci-lint:{{ golangci_lint_version }} golangci-lint run ./...

# Tidy go.mod and go.sum, and fail if that changes them
[group('check')]
tidy:
    go mod tidy
    git diff --exit-code -- go.mod go.sum

# Dependabot proposes these updates every week; this is the same list on demand.
# The braces are doubled because single ones are just's own.

# List the direct dependencies that have a newer release
[group('check')]
outdated:
    #!/bin/sh
    set -eu
    updates=$(go list -m -u -f '{{{{if and (not .Indirect) .Update}}{{{{.Path}} {{{{.Version}} -> {{{{.Update.Version}}{{{{end}}' all)
    if [ -n "$updates" ]; then
      echo "$updates"
    else
      echo "Every direct dependency is at its newest release."
    fi

# A test fails while docs/USAGE.md differs from what the program prints.

# Regenerate docs/USAGE.md from the program's help, after changing a command
[group('docs')]
docs:
    go run ./cmd/ffcc-gci help -markdown > docs/USAGE.md.tmp
    mv docs/USAGE.md.tmp docs/USAGE.md

# Check that CHANGELOG.md follows the Keep a Changelog format
[group('release')]
changelog-check:
    sh scripts/changelog.sh check

# Print the notes CHANGELOG.md holds for VERSION: just release-notes 1.0.0
[arg('version', pattern='[0-9]+\.[0-9]+\.[0-9]+', help='a released version, without the v')]
[group('release')]
release-notes version:
    @sh scripts/changelog.sh notes {{ version }}

# A push to main releases the newest version in CHANGELOG.md that has no tag
# yet. This is the same decision the workflow takes.

# Say what a push to main would release
[group('release')]
release-status:
    #!/bin/sh
    set -eu
    latest=$(sh scripts/changelog.sh latest)
    if [ -z "$latest" ]; then
      echo "CHANGELOG.md has no released version yet: a push to main releases nothing."
    elif git rev-parse -q --verify "refs/tags/v$latest" >/dev/null; then
      echo "v$latest is already tagged: a push to main releases nothing."
    else
      echo "v$latest has no tag yet: a push to main releases it, with these notes:"
      echo
      sh scripts/changelog.sh notes "$latest"
    fi
