ROOT_DIR := $(shell pwd)


bin:
	mkdir -p bin

build: bin
	go build -C cmd/pulumictl -o ${ROOT_DIR}/bin

install:
	go install -C cmd/pulumictl

clean:
	rm -rf bin

lint:
	cd cmd && golangci-lint run -c ../.golangci.yml --timeout 5m

lint_docker:
	mkdir -p .golangci-docker-cache
	docker run --rm \
		-v $$(pwd):/app \
		-v $$(pwd)/.golangci-docker-cache:/root/.cache/golangci-lint \
		-w /app \
		golangci/golangci-lint:v1.64.2 \
		golangci-lint run -c .golangci.yml --timeout 5m ./cmd/...
