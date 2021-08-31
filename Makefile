ROOT_DIR := $(shell pwd)



build:
	cd cmd/pulumictl && go build
	mv cmd/pulumictl/pulumictl ${ROOT_DIR}/

install:
	cd cmd/pulumictl && go install

clean:
	rm -f pulumictl

lint:
	cd cmd && golangci-lint run -c ../.golangci.yml --timeout 5m
