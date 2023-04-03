#!make

.PHONY: build
build:
	CGO_ENABLED=0 go build -v -o ./bin/http-chat-client ./http-chat-client.go

.PHONY: run
run: build
	HTTPCHAT_USERNAME=user1 CONFIG_FILENAME=client.cfg ./bin/http-chat-client
