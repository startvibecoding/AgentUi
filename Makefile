GO ?= $(shell if command -v go >/dev/null 2>&1; then command -v go; elif [ -x /home/free/usr/go/bin/go ]; then printf /home/free/usr/go/bin/go; else printf go; fi)
PKGS ?= ./...
EXAMPLE ?= ./examples/minimal
GOCACHE ?= /tmp/agentui-go-build-cache

.DEFAULT_GOAL := help

.PHONY: help fmt test vet race cross check ci run tidy clean

help:
	@printf "agentui make targets:\n"
	@printf "  make fmt     Format Go packages\n"
	@printf "  make test    Run unit tests\n"
	@printf "  make vet     Run go vet\n"
	@printf "  make race    Run tests with the race detector\n"
	@printf "  make cross   Compile-test Linux, macOS, and Windows targets\n"
	@printf "  make check   Run fmt, vet, and tests\n"
	@printf "  make ci      Run vet, tests, and race tests\n"
	@printf "  make run     Run the minimal example\n"
	@printf "  make tidy    Tidy go.mod/go.sum\n"
	@printf "  make clean   Clear Go build and test caches\n"

fmt:
	GOCACHE=$(GOCACHE) $(GO) fmt $(PKGS)

test:
	GOCACHE=$(GOCACHE) $(GO) test $(PKGS)

vet:
	GOCACHE=$(GOCACHE) $(GO) vet $(PKGS)

race:
	GOCACHE=$(GOCACHE) $(GO) test -race $(PKGS)

cross:
	GOCACHE=$(GOCACHE) GOOS=linux GOARCH=amd64 $(GO) test -exec=/bin/true $(PKGS)
	GOCACHE=$(GOCACHE) GOOS=darwin GOARCH=amd64 $(GO) test -exec=/bin/true $(PKGS)
	GOCACHE=$(GOCACHE) GOOS=darwin GOARCH=arm64 $(GO) test -exec=/bin/true $(PKGS)
	GOCACHE=$(GOCACHE) GOOS=windows GOARCH=amd64 $(GO) test -exec=/bin/true $(PKGS)

check: fmt vet test

ci: vet test race

run:
	GOCACHE=$(GOCACHE) $(GO) run $(EXAMPLE)

tidy:
	GOCACHE=$(GOCACHE) $(GO) mod tidy

clean:
	GOCACHE=$(GOCACHE) $(GO) clean -cache -testcache
