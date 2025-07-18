DEBUG ?= "debug"

# HOSTARCH is the host architecture
# ARCH is the target architecture
# we need to keep track of them separately
HOSTARCH ?= $(shell uname -m)
HOSTOS ?= $(shell uname -s | tr A-Z a-z)

# canonicalized names for host architecture
override HOSTARCH := $(subst aarch64,arm64,$(subst x86_64,amd64,$(HOSTARCH)))

# unless otherwise set, I am building for my own architecture, i.e. not cross-compiling
# and for my OS
ARCH ?= $(HOSTARCH)
OS ?= $(HOSTOS)

# canonicalized names for target architecture
override ARCH := $(subst aarch64,arm64,$(subst x86_64,amd64,$(ARCH)))

WORKDIR ?= $(CURDIR)/../../dist
TESTDIR := tests/$(shell basename $(CURDIR))
BINDIR := $(WORKDIR)/bin
DATADIR := $(WORKDIR)/$(TESTDIR)/
BIN := eden
LOCALBIN := $(BINDIR)/$(BIN)-$(OS)-$(ARCH)
TESTNAME := eden.kubevirt
TESTBIN := $(TESTNAME).test
TESTSCN := $(TESTNAME).tests.txt
LOCALTESTBIN := $(TESTBIN)-$(OS)-$(ARCH)
LINKDIR := ../../tests/kubevirt

.DEFAULT_GOAL := help

clean:
	rm -rf $(LOCALTESTBIN) $(BINDIR)/$(TESTBIN) $(WORKDIR)/$(TESTSCN) $(CURDIR)/$(TESTBIN) $(BINDIR)/$(TESTBIN)

$(BINDIR):
	mkdir -p $@
$(DATADIR):
	mkdir -p $@

test_kubevirt:
	go test kubevirt_test.go common.go -v -count=1 -timeout 3000s

test:
	$(LOCALBIN) test $(CURDIR) -v $(DEBUG) -s eden.kubevirt.tests.txt

build: setup

testbin: $(TESTBIN)
$(LOCALTESTBIN): $(BINDIR) *.go
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go test -c -ldflags "-s -w" -o $@ *.go

$(TESTBIN): $(LOCALTESTBIN)
	ln -sf $(LOCALTESTBIN) $(CURDIR)/$(TESTBIN)

setup: testbin $(BINDIR) $(DATADIR)
	cp -a $(LOCALTESTBIN) $(CURDIR)/$(TESTBIN) $(BINDIR)
	cp -a *.yml $(TESTSCN) $(DATADIR)

.PHONY: test build setup clean all testbin

help:
	@echo "EDEN is the harness for testing EVE and ADAM"
	@echo
	@echo "This Makefile automates commons tasks of EDEN testing"
	@echo
	@echo "Commonly used maintenance and development targets:"
	@echo "   build         build test-binary (OS and ARCH options supported, for ex. OS=linux ARCH=arm64)"
	@echo "   setup         setup of test environment"
	@echo "   test          run tests"
	@echo "   clean         cleanup of test harness"
	@echo
	@echo "You need install requirements for EVE (look at https://github.com/lf-edge/eve#install-dependencies)."
	@echo "You need access to docker socket and installed qemu packages."
