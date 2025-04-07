GO=CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go
BIN=pingdom-exporter
IMAGE=kokuwaio/$(BIN)
DOCKER_BIN=docker

SRC_DIR = .
GOFMT := gofmt
GOFMT_FLAGS := -s -w

TAG=$(shell git describe --tags)

.PHONY: build
build:
	$(GO) build -a --ldflags "-X main.VERSION=$(TAG) -w -extldflags '-static'" -tags netgo -o bin/$(BIN) ./cmd/$(BIN)

.PHONY: test
test:
	go vet $(SRC_DIR)/...
	go test -coverprofile=coverage.out $(SRC_DIR)/...
	go tool cover -func=coverage.out

.PHONY: lint
lint:
	go install golang.org/x/lint/golint@latest
	golint $(SRC_DIR)/...

# Build the Docker build stage TARGET
.PHONY: image
image:
	$(DOCKER_BIN) build -t $(IMAGE):$(TAG) $(SRC_DIR)

# Push Docker images to the registry
.PHONY: publish
publish:
	$(DOCKER_BIN) push $(IMAGE):$(TAG)
	$(DOCKER_BIN) tag $(IMAGE):$(TAG) $(IMAGE):latest
	$(DOCKER_BIN) push $(IMAGE):latest

fmt:
	$(GOFMT) $(GOFMT_FLAGS) $(SRC_DIR)
