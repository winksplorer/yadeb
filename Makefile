# I am addicted to gmake.

APP = yadeb
VERSION = 0.2

PREFIX ?= /usr/local

GO ?= go
GOFLAGS ?= -buildvcs=false -trimpath
GO_LDFLAGS ?= -s -w -buildid= -X main.BuildDate=$(shell date +%Y-%b-%d) -X main.Version=$(VERSION)

ifeq ($(SMALL),1)
	GOFLAGS += -gcflags=all=-l
endif

.PHONY: all install

all:
	$(GO) mod tidy
	$(GO) build -o $(APP) $(GOFLAGS) -ldflags="$(GO_LDFLAGS)"

install:
	install -m 755 $(APP) $(PREFIX)/bin