# Makefile for boxl

BINARY=boxl
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean

.PHONY: all build test clean

all: build

build:
	$(GOBUILD) -o $(BINARY) ./cmd/boxl

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY)
