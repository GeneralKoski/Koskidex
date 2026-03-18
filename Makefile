.PHONY: build test run clean docker docker-run lint bench

# Output binary name
BINARY=koskidex
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags="-X main.version=$(VERSION)" -o $(BINARY) main.go

test:
	go test -v ./...

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
	golangci-lint run ./...

bench:
	go test -bench=. -benchmem ./internal/engine/
