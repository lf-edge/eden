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
TESTBIN := eden.integration.test
LOCALTESTBIN := $(BINDIR)/$(TESTBIN)-$(OS)-$(ARCH)

MEDIA_SIZE=1024M

ZARCH ?= $(HOSTARCH)
export ZARCH

.DEFAULT_GOAL := help

clean: bin
	$(LOCALBIN) clean
	rm -rf $(LOCALBIN) $(BINDIR)/$(BIN) $(LOCALTESTBIN) $(BINDIR)/$(TESTBIN)

$(WORKDIR):
	mkdir -p $@
$(BINDIR):
	mkdir -p $@

test_controller:
	go test ./tests/integration/controller_test.go ./tests/integration/common.go -v -count=1 -timeout 3000s

test_base_image: test_controller
	go test ./tests/integration/baseImage_test.go ./tests/integration/common.go -v -count=1 -timeout 4500s

test_network_instance: test_controller
	go test ./tests/integration/networkInstance_test.go ./tests/integration/common.go -v -count=1 -timeout 4000s

test_application_instance: test_controller
	go test ./tests/integration/application_test.go ./tests/integration/common.go -v -count=1 -timeout 4000s

test_hooks: test_controller
	go test ./tests/integration/hooks_test.go ./tests/integration/common.go -v -count=1 -timeout 4000s

test: test_base_image test_network_instance test_application_instance

build: bin testbin

bin: $(BIN)
$(LOCALBIN): $(BINDIR) cmd/*.go pkg/*/*.go pkg/*/*/*.go
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -o $@ .
$(BIN): $(LOCALBIN)
	@if [ "$(OS)" = "$(HOSTOS)" -a "$(ARCH)" = "$(HOSTARCH)" -a ! -e "$@" ]; then ln -sf $(LOCALBIN) $(BINDIR)/$@; fi
	@if [ "$(OS)" = "$(HOSTOS)" -a "$(ARCH)" = "$(HOSTARCH)" -a ! -e "$@" ]; then ln -sf $(LOCALBIN) $@; fi

testbin: $(TESTBIN)
$(LOCALTESTBIN): $(BINDIR) tests/integration/*.go
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go test -c -o $@ tests/integration/*.go
$(TESTBIN): $(LOCALTESTBIN)
	@if [ "$(OS)" = "$(HOSTOS)" -a "$(ARCH)" = "$(HOSTARCH)" -a ! -e "$@" ]; then ln -sf $(LOCALTESTBIN) $(BINDIR)/$@; fi

config: build
	$(LOCALBIN) config add -v debug

setup: config
	$(LOCALBIN) setup -v debug

run: setup
	$(LOCALBIN) start -v debug

stop: bin
	$(LOCALBIN) stop -v debug

download: bin
	$(LOCALBIN) download --output=$(EVE_DIST)/dist/$(ZARCH) --arch=$(ZARCH) --tag=$(EVE_REF)

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
	@echo "   download      download eve from docker"
	@echo
	@echo "You need install requirements for EVE (look at https://github.com/lf-edge/eve#install-dependencies)."
	@echo "Also, you need to install 'uuidgen' utility."
	@echo "You need access to docker socket and installed qemu packages."

