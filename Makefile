ROOT_DIR := $(shell pwd)


bin:
	mkdir -p bin

build: bin
	CGO_ENABLED=0 go build -C cmd/pulumictl -o ${ROOT_DIR}/bin

install:
	CGO_ENABLED=0 go install -C cmd/pulumictl

clean:
	rm -rf bin

lint:
	cd cmd && golangci-lint run -c ../.golangci.yml --timeout 5m
