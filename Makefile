VERSION = $(shell git describe --always)
GOFLAGS = -ldflags="-X main.Version=$(VERSION) -s -w"

.PHONY: default server deploy help

default: help

server:
	test -n "$(GOOS)" # GOOS
	test -n "$(GOARCH)" # GOARCH
	go build $(GOFLAGS)
	mkdir -p bin/server
	mv server bin/server/$(GOOS)-$(GOARCH)

deploy:
	GOOS=linux GOARCH=amd64 make server
	GOOS=darwin GOARCH=amd64 make server
	mkdir -p release/server
	rm -rf release/server/*
	@JFROG_CLI_OFFER_CONFIG=false jfrog bt dlv --user=rferrazz --key=$(BINTRAY_API_KEY) rferrazz/IO-Something/server/rolling release/
	go-selfupdate -o release/server bin/server $(VERSION)
	@cd release && JFROG_CLI_OFFER_CONFIG=false jfrog bt u --user=rferrazz --key=$(BINTRAY_API_KEY) --override=true --flat=false --publish=true server/ rferrazz/IO-Something/server/rolling

clean:
	git clean -dfx

help:
	@echo "make [server clean deploy]"
