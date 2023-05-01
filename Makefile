GO		?= go
DOCKER		?= docker
DOCKER_BUILDKIT ?= 1
GIT_HASH	?= $(shell git log --pretty=format:%h -n 1)
BUILD_TIME	?= $(shell date)
# -s removes symbol table and -ldflags -w debugging symbols
LDFLAGS		?= -asmflags -trimpath -ldflags \
		   "-s -w -X 'main.gitHash=${GIT_HASH}' -X 'main.buildTime=${BUILD_TIME}'"
GOARCH		?= amd64
GOOS		?= linux
# CGO_ENABLED=0 == static by default
CGO_ENABLED	?= 0


all: test lint build

build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		$(GO) build $(LDFLAGS) \
		-o dist/$(APP_NAME) \
		main.go

.PHONY: clean
clean:
	rm -rf dist/

install-dependencies:
	@go get -d -v ./...

lint:
	@golangci-lint run ./...

vulncheck:
	@govulncheck -v ./...

escape-analysis:
	$(GO) build -gcflags="-m" 2>&1

docker-build:
	@DOCKER_BUILDKIT=$(DOCKER_BUILDKIT) $(DOCKER) \
			build --rm --target app -t $(APP_NAME)-builder .

docker-get-artifact:
	$(DOCKER) create -ti --name tmp $(APP_NAME)-builder /bin/bash
	-mkdir dist/
	$(DOCKER) cp tmp:/go/src/app/dist/$(APP_NAME) dist/$(APP_NAME)
	$(DOCKER) rm -f tmp

build-artifact: docker-build docker-get-artifact

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test:
	go test ./...

# This runs all tests, including integration tests
test-integration: start-db
	-@go test -tags=integration ./...
	@docker compose down

.PHONY: sqlc
sqlc:
	sqlc generate
