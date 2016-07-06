CWD=$(shell pwd)
VENDORGOPATH := $(CWD)/vendor:$(CWD)
GOPATH := $(CWD)

prep:
	if test -d pkg; then rm -rf pkg; fi

self:   prep
	if test -d src/github.com/whosonfirst/go-httpony; then rm -rf src/github.com/whosonfirst/go-httpony; fi
	mkdir -p src/github.com/whosonfirst/go-httpony
	cp httpony.go src/github.com/whosonfirst/go-httpony/
	cp -r cors src/github.com/whosonfirst/go-httpony/
	cp -r tls src/github.com/whosonfirst/go-httpony/
	cp -r rewrite src/github.com/whosonfirst/go-httpony/
	cp -r crumb src/github.com/whosonfirst/go-httpony/
	cp -r crypto src/github.com/whosonfirst/go-httpony/
	cp -r sso src/github.com/whosonfirst/go-httpony/

rmdeps:
	if test -d src; then rm -rf src; fi 

build:	rmdeps deps bin

deps:   
	@GOPATH=$(GOPATH) go get -u "github.com/vaughan0/go-ini"
	@GOPATH=$(GOPATH) go get -u "golang.org/x/net/html"
	@GOPATH=$(GOPATH) go get -u "golang.org/x/oauth2"

fmt:
	go fmt cmd/*.go
	go fmt *.go
	go fmt cors/*.go
	go fmt crumb/*.go
	go fmt tls/*.go
	go fmt rewrite/*.go
	go fmt sso/*.go

bin: 	self fmt
	@GOPATH=$(GOPATH) go build -o bin/echo-pony cmd/echo-pony.go
