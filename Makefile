ROOT_DIR := $(shell pwd)


bin:
	mkdir -p bin

build: bin
	go build -C cmd/pulumictl -o ${ROOT_DIR}/bin

install:
	go -C cmd/pulumictl install

clean:
	rm -rf bin

lint:
	cd cmd && golangci-lint run -c ../.golangci.yml --timeout 5m
