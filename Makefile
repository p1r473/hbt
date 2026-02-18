.PHONY: help
help: ## Show this
	@grep -E '^[0-9a-zA-Z_-]+:(.*?## .*|[a-z _0-9]+)?$$' Makefile | sort | awk 'BEGIN {FS = ":(.*## |[\t ]*)"}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

NAME:=hbtsrv
build_tag:=$(shell git describe --tags 2> /dev/null)
BUILDFLAGS:="-s -w -X github.com/lzambarda/hbt/internal/config.Version=$(build_tag)"

.PHONY: dependencies
dependencies: ## Install dependencies requried for development operations.
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@go mod tidy


.PHONY: lint
lint:
	go fmt ./...
	golangci-lint run ./...


.PHONY: build
build:
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -ldflags $(BUILDFLAGS) -o ./bin/darwin/$(NAME) ./cmd/server/*.go
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -ldflags $(BUILDFLAGS) -o bin/linux/$(NAME) ./cmd/server/*.go


.PHONY: build_assets
build_assets: build
	@mkdir -p assets
	@tar -zcvf assets/darwin-amd64-$(NAME).tgz ./bin/darwin/$(NAME)
	@tar -zcvf assets/linux-amd64-$(NAME).tgz ./bin/linux/$(NAME)

run="."
dir="./..."
short="-short"
.PHONY: test
test:
	@go test --timeout=40s $(short) $(dir) -run $(run);
