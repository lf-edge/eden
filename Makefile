DEBUG ?= "debug"
CONFIG ?=
TESTS ?= $(shell find tests/ -maxdepth 1 -mindepth 1 -type d  -exec basename {} \;)
DO_DOCKER ?= 1

DOCKER_TARGET ?= load
DOCKER_PLATFORM ?= $(shell uname -s | tr '[A-Z]' '[a-z]')/$(subst aarch64,arm64,$(subst x86_64,amd64,$(shell uname -m)))

# ESERVER_TAG is the tag for eserver image to build
ESERVER_TAG ?= "lfedge/eden-http-server"
# ESERVER_DIR is the directory with eserver Dockerfile to build
ESERVER_DIR=$(CURDIR)/eserver
# ESERVER_VERSION is the version of eserver image to build
ESERVER_VERSION ?= $(shell git rev-parse --short HEAD:eserver)

# PROCESSING_TAG is the tag for processing image to build
PROCESSING_TAG ?= "lfedge/eden-processing"
# PROCESSING_DIR is the directory with processing Dockerfile to build
PROCESSING_DIR=$(CURDIR)/processing
# PROCESSING_VERSION is the version of processing image to build
PROCESSING_VERSION ?= $(shell git tag -l --contains HEAD)
ifeq ($(PROCESSING_VERSION),)
	PROCESSING_VERSION = $(shell git describe --always)
endif


# EDEN_TAG is the tag for eden image to build
EDEN_TAG ?= "lfedge/eden"
# EDEN_VERSION is the version of eden image to build
EDEN_VERSION ?= $(shell git tag -l --contains HEAD)
ifeq ($(EDEN_VERSION),)
	EDEN_VERSION = $(shell git describe --always)
endif

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
EMPTY_DRIVE := $(WORKDIR)/empty
EMPTY_DRIVE_SIZE := 10M

DIRECTORY_EXPORT ?= $(CURDIR)/export

ZARCH ?= $(HOSTARCH)
export ZARCH

LINUXKIT=$(WORKDIR)/bin/linuxkit
LINUXKIT_VERSION=fc060cac15e6bb3b5977c78996a0a7bd60b4a024
LINUXKIT_SOURCE=https://github.com/linuxkit/linuxkit.git

.DEFAULT_GOAL := help

clean: config stop
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) clean
	$(LOCALBIN) clean --current-context=false
	rm -rf $(LOCALBIN) $(BINDIR)/$(BIN) $(LOCALTESTBIN) $(WORKDIR)

$(WORKDIR):
	mkdir -p $@

$(BINDIR):
	mkdir -p $@

$(DIRECTORY_EXPORT):
	mkdir -p $@

test: build-tests
	make -C tests TESTS="$(TESTS)" DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) test

unit-test:
	go test $(go list ./... | grep -v /eden/tests/)

# create empty drives to use as additional volumes
$(EMPTY_DRIVE).%:
	qemu-img create -f $* $@ $(EMPTY_DRIVE_SIZE)

build-tests: build testbin
install: build
	CGO_ENABLED=0 go install .

build: $(BIN) $(EMPTY_DRIVE).raw $(EMPTY_DRIVE).qcow2 $(EMPTY_DRIVE).qcow $(EMPTY_DRIVE).vmdk $(EMPTY_DRIVE).vhdx $(LINUXKIT)
$(LOCALBIN): $(BINDIR) cmd/*.go pkg/*/*.go pkg/*/*/*.go
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -ldflags "-s -w" -o $@ .
	mkdir -p dist/scripts/shell
	cp -r shell-scripts/* dist/scripts/shell/

build-tools: $(LINUXKIT)
	@echo Done building $<

$(BIN): $(LOCALBIN)
	ln -sf $(BIN)-$(OS)-$(ARCH) $(BINDIR)/$@
	ln -sf $(LOCALBIN) $@
	ln -sf bin/$@ $(WORKDIR)/$@

$(LINUXKIT):
	@rm -rf /tmp/linuxkit
	@git clone $(LINUXKIT_SOURCE) /tmp/linuxkit
	@cd /tmp/linuxkit && git checkout $(LINUXKIT_VERSION)
	@cd /tmp/linuxkit/src/cmd/linuxkit && GO111MODULE=on CGO_ENABLED=0 go build -o $@ -mod=vendor .
	@rm -rf /tmp/linuxkit

testbin: config
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) build

config: build
ifeq ($(OS), $(HOSTOS))
	$(LOCALBIN) config add default -v $(DEBUG) $(CONFIG)
endif

setup: config build-tests
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) setup
	$(LOCALBIN) setup -v $(DEBUG)

run: build setup
	$(LOCALBIN) start -v $(DEBUG)

stop: build
	$(LOCALBIN) stop -v $(DEBUG)

dist: build-tests
	tar cvzf dist/eden_dist.tgz dist/bin dist/scripts dist/tests dist/*.txt

.PHONY: all clean test build build-tests tests-export config setup stop testbin dist

push-multi-arch-eserver:
	@echo "Build and $(DOCKER_TARGET) eserver image $(ESERVER_TAG):$(ESERVER_VERSION)"
	@docker buildx build --$(DOCKER_TARGET) --platform $(DOCKER_PLATFORM) --tag $(ESERVER_TAG):$(ESERVER_VERSION) $(ESERVER_DIR)

push-multi-arch-eden:
	@echo "Build and $(DOCKER_TARGET) eden image $(EDEN_TAG):$(EDEN_VERSION)"
	@docker buildx build --$(DOCKER_TARGET) --platform $(DOCKER_PLATFORM) --tag $(EDEN_TAG):$(EDEN_VERSION) .

push-multi-arch-processing:
	@echo "Build and $(DOCKER_TARGET) processing image $(PROCESSING_TAG):$(PROCESSING_VERSION)"
	@docker buildx build --$(DOCKER_TARGET) --platform $(DOCKER_PLATFORM) --tag $(PROCESSING_TAG):$(PROCESSING_VERSION) $(PROCESSING_DIR)

build-docker: push-multi-arch-processing push-multi-arch-eserver push-multi-arch-eden
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) DOCKER_TARGET=$(DOCKER_TARGET) DOCKER_PLATFORM=$(DOCKER_PLATFORM) build-docker

tests-export: $(DIRECTORY_EXPORT) build-tests
	@cp -af $(WORKDIR)/tests/* $(DIRECTORY_EXPORT)
	@echo "Your tests inside $(DIRECTORY_EXPORT)"

yetus:
	@echo Running yetus
	docker run -it --rm -v $(CURDIR):/src:delegated -v /tmp:/tmp apache/yetus:0.14.0 \
		--basedir=/src \
		--dirty-workspace \
		--empty-patch \
		--plugins=all

validate:
	@echo Running static validation checks...
	@echo ...on model files
	@tar -cf - models/*.json | docker run -i alpine sh -c \
		'tar xf - && apk add jq >&2 && for i in models/*.json; do echo "$$i" >&2 && jq -r ".logo | to_entries[] | .value" "$$i" || exit 1; done' |\
		while read logo; do echo "$$logo" ; if [ ! -f models/`basename "$$logo"` ]; then echo "can't find $$logo" && exit 1; fi; done


help:
	@echo "EDEN is the harness for testing EVE and ADAM"
	@echo
	@echo "This Makefile automates commons tasks of building and running"
	@echo "  * EVE"
	@echo "  * ADAM"
	@echo
	@echo "Commonly used maintenance and development targets:"
	@echo "   dist          make distribution archive dist/eden_dist.tgz"
	@echo "   run           run ADAM and EVE"
	@echo "   test          run tests"
	@echo "   config        generate required config files"
	@echo "   setup         download and/or build required files"
	@echo "   stop          stop ADAM and EVE"
	@echo "   clean         full cleanup of test harness"
	@echo "   build         build utilities (OS and ARCH options supported, for ex. OS=linux ARCH=arm64)"
	@echo "   build-docker  build all docker images of EDEN"
	@echo
	@echo "You can use some parameters:"
	@echo "   CONFIG        additional parameters for 'eden config add default', for ex. \"make CONFIG='--devmodel RPi4' run\" or \"make CONFIG='--devmodel GCP' run\""
	@echo "   TESTS         list of tests for 'make test' to run, for ex. make TESTS='lim units' test"
	@echo "   DEBUG         debug level for 'eden' command ('debug' by default)"
	@echo "yetus            run Apache Yetus to check the quality of the source tree"
	@echo "tests-export     exports escripts into export directory, content of export directory should be inside tests directory in root of another repo"
	@echo
	@echo "You need install requirements for EVE (look at https://github.com/lf-edge/eve#install-dependencies)."
	@echo "You need access to docker socket and installed qemu packages."
