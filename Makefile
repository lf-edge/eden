DEBUG ?= "debug"
CONFIG ?= 
TESTS ?= $(shell find tests/ -maxdepth 1 -mindepth 1 -type d  -exec basename {} \;)

# ESERVER_TAG is the tag for eserver image to build
ESERVER_TAG ?= "lfedge/eden-http-server"
# ESERVER_VERSION is the version of eserver image to build
ESERVER_VERSION ?= "latest"
# ESERVER_DIR is the directory with eserver Dockerfile to build
ESERVER_DIR=$(CURDIR)/eserver
# check if eserver image already exists in local docker and get its IMAGE_ID
ESERVER_IMAGE_ID ?= $(shell docker images -q $(ESERVER_TAG):$(ESERVER_VERSION))

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

WORKDIR=$(CURDIR)/dist
BINDIR := $(WORKDIR)/bin
BIN := eden
LOCALBIN := $(BINDIR)/$(BIN)-$(OS)-$(ARCH)
EMPTY_DRIVE := $(WORKDIR)/empty.qcow2

ZARCH ?= $(HOSTARCH)
export ZARCH

.DEFAULT_GOAL := help

clean: config stop
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) clean
	$(LOCALBIN) clean
	rm -rf $(LOCALBIN) $(BINDIR)/$(BIN) $(LOCALTESTBIN) $(WORKDIR)

$(WORKDIR):
	mkdir -p $@

$(BINDIR):
	mkdir -p $@

test: build
	make -C tests TESTS="$(TESTS)" DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) test

# create empty drive in qcow2 format to use as additional volumes
$(EMPTY_DRIVE):
	qemu-img create -f qcow2 $(EMPTY_DRIVE) 100M

build: bin testbin
install: build
	CGO_ENABLED=0 go install .

bin: $(BIN) $(EMPTY_DRIVE)
ifeq ($(ESERVER_IMAGE_ID), ) # if we need to build eserver
bin: $(BIN) $(EMPTY_DRIVE) eserver
endif
$(LOCALBIN): $(BINDIR) cmd/*.go pkg/*/*.go pkg/*/*/*.go
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -o $@ .
$(BIN): $(LOCALBIN)
	@if [ "$(OS)" = "$(HOSTOS)" -a "$(ARCH)" = "$(HOSTARCH)" ]; then ln -sf $(LOCALBIN) $(BINDIR)/$@; fi
	@if [ "$(OS)" = "$(HOSTOS)" -a "$(ARCH)" = "$(HOSTARCH)" ]; then ln -sf $(LOCALBIN) $@; fi

testbin: config
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) build

config: bin
	$(LOCALBIN) config add default -v $(DEBUG) $(CONFIG)

setup: config
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) setup
	$(LOCALBIN) setup -v $(DEBUG)

run: bin setup
	$(LOCALBIN) start -v $(DEBUG)

stop: bin
	$(LOCALBIN) stop -v $(DEBUG)

.PHONY: eserver

eserver:
	@echo "Build eserver image"
	docker build -t $(ESERVER_TAG):$(ESERVER_VERSION) $(ESERVER_DIR)

help:
	@echo "EDEN is the harness for testing EVE and ADAM"
	@echo
	@echo "This Makefile automates commons tasks of building and running"
	@echo "  * EVE"
	@echo "  * ADAM"
	@echo
	@echo "Commonly used maintenance and development targets:"
	@echo "   run           run ADAM and EVE"
	@echo "   test          run tests"
	@echo "   config        generate required config files"
	@echo "   setup         download and/or build required files"
	@echo "   stop          stop ADAM and EVE"
	@echo "   clean         full cleanup of test harness"
	@echo "   build         build utilities (OS and ARCH options supported, for ex. OS=linux ARCH=arm64)"
	@echo "   eserver       build eserver image"
	@echo
	@echo "You can use some parameters:"
	@echo "   CONFIG        additional parameters for 'eden config add default', for ex. \"make CONFIG='--devmodel RPi4' run\" or \"make CONFIG='--devmodel GCP' run\""
	@echo "   TESTS         list of tests for 'make test' to run, for ex. make TESTS='lim units' test"
	@echo "   DEBUG         debug level for 'eden' command ('debug' by default)"
	@echo
	@echo "You need install requirements for EVE (look at https://github.com/lf-edge/eve#install-dependencies)."
	@echo "You need access to docker socket and installed qemu packages."

