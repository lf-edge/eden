DEBUG ?= "debug"
TESTS ?= $(wildcard */Makefile)

DOCKER_TARGET ?= load
DOCKER_PLATFORM ?= $(shell uname -s | tr '[A-Z]' '[a-z]')/$(subst aarch64,arm64,$(subst x86_64,amd64,$(shell uname -m)))

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

WORKDIR ?= $(CURDIR)/../dist

gotestsum:
	go get -d gotest.tools/gotestsum

#test: $(TESTS:=_test)
test: gotestsum
	gotestsum --jsonfile $(WORKDIR)/results.json --junitfile $(WORKDIR)/results.xml --raw-command -- go tool test2json -t ../eden test workflow -v debug

build: $(TESTS:=_build)
setup: $(TESTS:=_setup)
clean: $(TESTS:=_clean)
build-docker: $(TESTS:=_build_docker)

.PHONY: test build setup clean all

%_test: % %_build %_setup
	#make -C $$(dirname $<) DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) test

%_build: %
	make -C $$(dirname $<) DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) build

%_setup: %
	make -C $$(dirname $<) DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) setup

%_clean: %
	make -C $$(dirname $<) DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) clean

%_build_docker: %
	@grep ^build-docker: $< && make -C $$(dirname $<) DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) DOCKER_TARGET=$(DOCKER_TARGET) DOCKER_PLATFORM=$(DOCKER_PLATFORM) build-docker; exit 0
