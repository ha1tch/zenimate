# zenimate - Makefile front-end
#
# The GUI links raylib. Two build modes:
#   * purego (default here): cgo-free, no system GL headers needed to build.
#   * cgo:    needs system dev libraries (see README) but is the normal runtime
#             path on a desktop.

VERSION := $(shell tr -d ' \t\r\n' < VERSION)
DIST    := dist
GUITAGS := purego

GO      ?= go

.DEFAULT_GOAL := help

## help: show this help
.PHONY: help
help:
	@echo "zenimate v$(VERSION) - targets:"
	@echo
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
	@echo

## build: build the GUI frontend into $(DIST)
.PHONY: build
build: build-gui

## build-gui: build the raylib frontend (cgo-free purego path)
.PHONY: build-gui
build-gui:
	@mkdir -p $(DIST)
	CGO_ENABLED=0 $(GO) build -tags $(GUITAGS) -o $(DIST)/zenimate-gui ./cmd/zenimate-gui/

## build-gui-cgo: build the raylib frontend via cgo (needs system GL/X11/Wayland dev libs)
.PHONY: build-gui-cgo
build-gui-cgo:
	@mkdir -p $(DIST)
	$(GO) build -o $(DIST)/zenimate-gui ./cmd/zenimate-gui/

## test: run the logic test suite and the purego GUI tests
.PHONY: test
test:
	$(GO) test ./pkg/... ./internal/... -count=1
	CGO_ENABLED=0 $(GO) test -tags $(GUITAGS) ./cmd/zenimate-gui/ -count=1

## race: run the test suite under the race detector
.PHONY: race
race:
	$(GO) test -race ./pkg/... ./internal/... -count=1

## vet: run go vet on all packages (GUI uses the purego tag)
.PHONY: vet
vet:
	$(GO) vet ./pkg/... ./internal/...
	CGO_ENABLED=0 $(GO) vet -tags $(GUITAGS) ./cmd/zenimate-gui/

## fmt: gofmt the whole tree in place
.PHONY: fmt
fmt:
	gofmt -w .

## fmt-check: fail if anything is not gofmt-clean
.PHONY: fmt-check
fmt-check:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt would change:"; echo "$$out"; exit 1; fi
	@echo "gofmt clean"

## cross: cross-compile the GUI for all release target platforms (to /dev/null)
.PHONY: cross
cross:
	@for p in darwin/amd64 linux/amd64 linux/arm64 windows/amd64; do \
		os=$${p%/*}; arch=$${p#*/}; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 $(GO) build -tags $(GUITAGS) -o /dev/null ./cmd/zenimate-gui/ && echo "ok  gui  $$p"; \
	done

## check: the full pre-commit gate (fmt-check, vet, cross, test)
.PHONY: check
check: fmt-check vet cross test

## sync: propagate VERSION into pkg/version
.PHONY: sync
sync:
	bash scripts/syncver.sh

## tidy: go mod tidy
.PHONY: tidy
tidy:
	$(GO) mod tidy

## release: run the full release pipeline (validate, build, test, checkpoint zip)
.PHONY: release
release:
	bash scripts/release.sh

## clean: remove build artifacts
.PHONY: clean
clean:
	rm -rf $(DIST)
	@echo "cleaned $(DIST)"
