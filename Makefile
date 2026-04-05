GO ?= go

.PHONY: test build doctor run-example

test:
	$(GO) test ./...

build:
	$(GO) build ./...

doctor:
	$(GO) run ./cmd/whisky doctor

run-example:
	$(GO) run ./examples/pixel-quest/cmd/game
