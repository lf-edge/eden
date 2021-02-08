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

# IMAGE_TAG is the tag for image to build
IMAGE_TAG ?= itmoeve/fio_tests
# IMAGE_VERSION is the version of image to build
IMAGE_VERSION ?= v.1.2
# IMAGE_DIR is the directory with image Dockerfile to build
IMAGE_DIR=$(CURDIR)/image

# canonicalized names for target architecture
override ARCH := $(subst aarch64,arm64,$(subst x86_64,amd64,$(ARCH)))

WORKDIR ?= $(CURDIR)/../../dist
TESTDIR := tests/$(shell basename $(CURDIR))
BINDIR := $(WORKDIR)/bin
DATADIR := $(WORKDIR)/$(TESTDIR)/
BIN := eden
LOCALBIN := $(BINDIR)/$(BIN)-$(OS)-$(ARCH)
TESTNAME := eden.fio
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

setup: $(BINDIR) $(DATADIR)
	cp -a *.yml $(TESTSCN) testdata $(DATADIR)

.PHONY: image

image:
	@echo "Build image"
	docker build -t $(IMAGE_TAG):$(IMAGE_VERSION) $(IMAGE_DIR)

push-multi-arch:
	@echo "Push image"
	docker buildx build --push --platform linux/arm64/v8,linux/amd64 --tag $(IMAGE_TAG):$(IMAGE_VERSION) $(IMAGE_DIR)

.PHONY: test build setup clean all

help:
	@echo "EDEN is the harness for testing EVE and ADAM"
	@echo
	@echo "This Makefile automates commons tasks of EDEN testing"
	@echo
	@echo "Commonly used maintenance and development targets:"
	@echo "   build         build test-binary (OS and ARCH options supported, for ex. OS=linux ARCH=arm64)"
	@echo "   image         build image for upload into docker"
	@echo "   setup         setup of test environment"
	@echo "   test          run tests"
	@echo "   clean         cleanup of test harness"
	@echo
	@echo "You need install requirements for EVE (look at https://github.com/lf-edge/eve#install-dependencies)."
	@echo "You need access to docker socket and installed qemu pac
