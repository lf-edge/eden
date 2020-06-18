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
TESTNAME := eden.integration
TESTBIN := $(TESTNAME).test
TESTSCN := $(TESTNAME).tests.txt
LOCALTESTBIN := $(TESTBIN)-$(OS)-$(ARCH)

IMAGESOURCES := images-src

.DEFAULT_GOAL := help

clean: 
	rm -rf $(LOCALTESTBIN) $(BINDIR)/$(TESTBIN) $(WORKDIR)/${IMAGESOURCES} $(WORKDIR)/$(TESTSCN) $(CURDIR)/$(TESTBIN) $(BINDIR)/$(TESTBIN)

$(WORKDIR):
	mkdir -p $@
$(BINDIR):
	mkdir -p $@

test_controller:
	go test controller_test.go common.go -v -count=1 -timeout 3000s

test_base_image: test_controller
	$(LOCALBIN) test -r TestBaseImage  -a "-baseos.eve.tag=571d94a11fa19d79805a0465030175b7257d343b" -v $(DEBUG)

test_network_instance: test_controller
	go test networkInstance_test.go common.go -v -count=1 -timeout 4000s

test_application_instance: test_controller
	$(LOCALBIN) test -r TestApplication -a "-app-docker.yml=${IMAGESOURCES}/docker/alpine/alpine.yml -app-vm.yml=${IMAGESOURCES}/vm/alpine/alpine.yml" -v $(DEBUG)

test: 
	$(LOCALBIN) test $(CURDIR) -v $(DEBUG)

build: setup

testbin: $(TESTBIN)
$(LOCALTESTBIN): $(BINDIR) *.go
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go test -c -o $@ *.go
$(TESTBIN): $(LOCALTESTBIN)
	@if [ "$(OS)" = "$(HOSTOS)" -a "$(ARCH)" = "$(HOSTARCH)" ]; then ln -sf $(LOCALTESTBIN) $(CURDIR)/$(TESTBIN); fi

setup: testbin ${IMAGESOURCES}
	cp $(TESTSCN) $(WORKDIR) 
	@if [ "$(OS)" = "$(HOSTOS)" -a "$(ARCH)" = "$(HOSTARCH)" ]; then ln -sf $(CURDIR)/$(TESTBIN) $(BINDIR)/; fi

${IMAGESOURCES}:
	cp -rf images $(WORKDIR)/${IMAGESOURCES}

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
	@echo "Also, you need to install 'uuidgen' utility."
	@echo "You need access to docker socket and installed qemu packages."

