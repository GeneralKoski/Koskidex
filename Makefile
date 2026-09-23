.PHONY: build test vet check run clean docker docker-run lint bench

# Output binary name
BINARY=koskidex
# ./... prenderebbe anche il package Go dentro web/node_modules.
PKGS := $(shell go list ./... | grep -v /node_modules/)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags="-X main.version=$(VERSION)" -o $(BINARY) main.go

test:
	go test -v $(PKGS)

vet:
	go vet $(PKGS)

# Controllo unico prima di ogni commit.
check: vet test

run: build
	./$(BINARY) --port 7700 --data-dir ./data

clean:
	rm -f $(BINARY)
	rm -rf ./data

docker:
	docker build -t koskidex .

docker-run: docker
	docker run --rm --name koskidex -p 7700:7700 -v $(PWD)/data:/data koskidex

lint:
	golangci-lint run $(PKGS)

bench:
	go test -bench=. -benchmem ./internal/engine/
