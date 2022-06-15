# Adapted from: https://gist.github.com/developer-guy/c73e5f003193ba120438c15ad0a75cd8

GIT_VERSION=$(shell git describe --tags --always --dirty)
GIT_REV=$(shell git rev-parse HEAD)
DATE_FMT = +'%Y-%m-%dT%H:%M:%SZ'
SOURCE_DATE_EPOCH ?= $(shell git log -1 --pretty=%ct)
ifdef SOURCE_DATE_EPOCH
    BUILD_DATE ?= $(shell date -u -d "@$(SOURCE_DATE_EPOCH)" "$(DATE_FMT)" 2>/dev/null || date -u -r "$(SOURCE_DATE_EPOCH)" "$(DATE_FMT)" 2>/dev/null || date -u "$(DATE_FMT)" 2>/dev/null)
else
    BUILD_DATE ?= $(shell date "$(DATE_FMT)")
endif

LDFLAGS="-X 'main.version=$(GIT_VERSION)' -X 'main.commit=$(GIT_REV)' -X main.date=$(BUILD_DATE)"

.PHONY: build snapshot release

build:
	go build -ldflags $(LDFLAGS) ./cmd/prom-k8s-versions

snapshot:
	LDFLAGS=$(LDFLAGS) goreleaser release --snapshot --rm-dist

release:
	export GITHUB_TOKEN=$(GITHUB_TOKEN) && \
	  LDFLAGS=$(LDFLAGS) goreleaser release --rm-dist
