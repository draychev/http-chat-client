#!make

.PHONY: build
build:
	CGO_ENABLED=0 go build -v -o ./bin/http-chat-client ./http-chat-client.go

.PHONY: run
run: build
	./bin/http-chat-client
