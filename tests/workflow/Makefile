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
BINDIR := $(WORKDIR)/bin
BIN := eden
LOCALBIN := $(BINDIR)/$(BIN)-$(OS)-$(ARCH)
TESTNAME := eden.workflow
TESTBIN := eden.escript.test
TESTSCN := $(TESTNAME).tests.txt
LOCALTESTBIN := $(TESTBIN)-$(OS)-$(ARCH)

.DEFAULT_GOAL := help

clean: 
	rm -rf $(WORKDIR)/$(TESTSCN) 

$(WORKDIR):
	mkdir -p $@

test:
	$(LOCALBIN) test $(CURDIR) -v $(DEBUG)

build: setup

setup: 
	cp $(TESTSCN) $(WORKDIR) 
	$(LOCALBIN) utils template eden-config.tmpl>eden-config.yml

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

