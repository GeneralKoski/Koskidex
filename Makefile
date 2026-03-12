.PHONY: build test run clean docker

# Output binary name
BINARY=koskidex

build:
	go build -o $(BINARY) main.go

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
