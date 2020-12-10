DEBUG ?= "debug"
CONFIG ?=
TESTS ?= $(shell find tests/ -maxdepth 1 -mindepth 1 -type d  -exec basename {} \;)
DO_DOCKER ?= 1

# ESERVER_TAG is the tag for eserver image to build
ESERVER_TAG ?= "lfedge/eden-http-server"
# ESERVER_VERSION is the version of eserver image to build
ESERVER_VERSION ?= "1.2"
# ESERVER_DIR is the directory with eserver Dockerfile to build
ESERVER_DIR=$(CURDIR)/eserver
# check if eserver image already exists in local docker and get its IMAGE_ID
ifeq ($(DO_DOCKER), 1) # if we need to build eserver
ESERVER_IMAGE_ID ?= $(shell docker images -q $(ESERVER_TAG):$(ESERVER_VERSION))
endif

# ESERVER_TAG is the tag for processing image to build
PROCESSING_TAG ?= "itmoeve/eden-processing"
# PROCESSING_VERSION is the version of processing image to build
PROCESSING_VERSION ?= "1.2"
# PROCESSING_DIR is the directory with processing Dockerfile to build
PROCESSING_DIR=$(CURDIR)/processing

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
BINDIR := dist/bin
BIN := eden
LOCALBIN := $(BINDIR)/$(BIN)-$(OS)-$(ARCH)
EMPTY_DRIVE_QCOW2 := $(WORKDIR)/empty.qcow2
EMPTY_DRIVE_RAW := $(WORKDIR)/empty.raw

ZARCH ?= $(HOSTARCH)
export ZARCH

.DEFAULT_GOAL := help

clean: config stop
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) clean
	$(LOCALBIN) clean --current-context=false
	rm -rf $(LOCALBIN) $(BINDIR)/$(BIN) $(LOCALTESTBIN) $(WORKDIR)

$(WORKDIR):
	mkdir -p $@

$(BINDIR):
	mkdir -p $@

test: build
	make -C tests TESTS="$(TESTS)" DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) test

# create empty drive in qcow2 format to use as additional volumes
$(EMPTY_DRIVE_QCOW2):
	qemu-img create -f qcow2 $(EMPTY_DRIVE_QCOW2) 100M

# create empty drive in raw format to use as additional volumes
$(EMPTY_DRIVE_RAW):
	qemu-img create -f raw $(EMPTY_DRIVE_RAW) 10M

build-tests: build testbin gotestsum
install: build
	CGO_ENABLED=0 go install .

build: $(BIN) $(EMPTY_DRIVE_RAW) $(EMPTY_DRIVE_QCOW2)
ifeq ($(ESERVER_IMAGE_ID), ) # if we need to build eserver
build: $(BIN) $(EMPTY_DRIVE_RAW) $(EMPTY_DRIVE_QCOW2) eserver
endif
$(LOCALBIN): $(BINDIR) cmd/*.go pkg/*/*.go pkg/*/*/*.go
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -o $@ .
	mkdir -p dist/scripts/shell
	cp shell-scripts/* dist/scripts/shell/

$(BIN): $(LOCALBIN)
	@if [ "$(OS)" = "$(HOSTOS)" -a "$(ARCH)" = "$(HOSTARCH)" ]; then ln -sf $(BIN)-$(OS)-$(ARCH) $(BINDIR)/$@; fi
	@if [ "$(OS)" = "$(HOSTOS)" -a "$(ARCH)" = "$(HOSTARCH)" ]; then ln -sf $(LOCALBIN) $@; fi

testbin: config
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) DO_TEMPLATE=$(DO_TEMPLATE) build

gotestsum:
	go get gotest.tools/gotestsum

config: build
	$(LOCALBIN) config add default -v $(DEBUG) $(CONFIG)

setup: config build-tests
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) setup
	$(LOCALBIN) setup -v $(DEBUG)

run: build setup
	$(LOCALBIN) start -v $(DEBUG)

stop: build
	$(LOCALBIN) stop -v $(DEBUG)

.PHONY: processing eserver all clean test build

eserver:
	@echo "Build eserver image"
	@if [ $(DO_DOCKER) -ne 0 ]; then docker build -t $(ESERVER_TAG):$(ESERVER_VERSION) $(ESERVER_DIR); fi

processing:
	@echo "Build processing image"
	@if [ $(DO_DOCKER) -ne 0 ]; then docker build -t $(PROCESSING_TAG):$(PROCESSING_VERSION) $(PROCESSING_DIR); fi

yetus:
	@echo Running yetus
	build-tools/src/yetus/test-patch.sh

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
	@echo "yetus          run Apache Yetus to check the quality of the source tree"
	@echo
	@echo "You need install requirements for EVE (look at https://github.com/lf-edge/eve#install-dependencies)."
	@echo "You need access to docker socket and installed qemu packages."
