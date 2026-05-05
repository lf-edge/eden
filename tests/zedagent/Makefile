DEBUG ?= "debug"

HOSTARCH ?= $(shell uname -m)
HOSTOS ?= $(shell uname -s | tr A-Z a-z)

override HOSTARCH := $(subst aarch64,arm64,$(subst x86_64,amd64,$(HOSTARCH)))

ARCH ?= $(HOSTARCH)
OS ?= $(HOSTOS)

override ARCH := $(subst aarch64,arm64,$(subst x86_64,amd64,$(ARCH)))

WORKDIR ?= $(CURDIR)/../../dist
TESTDIR := tests/$(shell basename $(CURDIR))
BINDIR := $(WORKDIR)/bin
DATADIR := $(WORKDIR)/$(TESTDIR)/
BIN := eden
LOCALBIN := $(BINDIR)/$(BIN)-$(OS)-$(ARCH)
TESTNAME := eden.zedagent
TESTBIN := $(TESTNAME).test
TESTSCN := $(TESTNAME).tests.txt
LOCALTESTBIN := $(TESTBIN)-$(OS)-$(ARCH)

.DEFAULT_GOAL := help

clean:
	rm -rf $(LOCALTESTBIN) $(BINDIR)/$(TESTBIN) $(WORKDIR)/$(TESTSCN) $(CURDIR)/$(TESTBIN) $(BINDIR)/$(TESTBIN)

$(BINDIR):
	mkdir -p $@
$(DATADIR):
	mkdir -p $@

test:
	$(LOCALBIN) test $(CURDIR) -v $(DEBUG)

build: setup

testbin: $(TESTBIN)
$(LOCALTESTBIN): $(BINDIR) *.go
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go test -c -ldflags "-s -w" -o $@ *.go

$(TESTBIN): $(LOCALTESTBIN)
	ln -sf $(LOCALTESTBIN) $(CURDIR)/$(TESTBIN)

setup: testbin $(BINDIR) $(DATADIR)
	cp -a $(LOCALTESTBIN) $(CURDIR)/$(TESTBIN) $(BINDIR)
	cp -a *.tests.txt testdata $(DATADIR)

.PHONY: test build setup clean all testbin

help:
	@echo "zedagent eden integration tests"
	@echo
	@echo "Targets:"
	@echo "   build   build test binary"
	@echo "   setup   build and stage test binary + data"
	@echo "   test    run tests against a running EVE instance"
	@echo "   clean   remove build artifacts"
