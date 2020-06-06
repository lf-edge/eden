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

WORKDIR=$(CURDIR)/dist
BINDIR := $(WORKDIR)/bin
BIN := eden
LOCALBIN := $(BINDIR)/$(BIN)-$(OS)-$(ARCH)

MEDIA_SIZE=1024M

ZARCH ?= $(HOSTARCH)
export ZARCH

.DEFAULT_GOAL := help

clean: stop
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) ECONFIG=$(ECONFIG) clean
	$(LOCALBIN) clean
	rm -rf $(LOCALBIN) $(BINDIR)/$(BIN) $(LOCALTESTBIN)

$(WORKDIR):
	mkdir -p $@

$(BINDIR):
	mkdir -p $@

ECONFIG := `$(LOCALBIN) config get`
test: build
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) ECONFIG=$(ECONFIG) test

build: bin testbin

bin: $(BIN)
$(LOCALBIN): $(BINDIR) cmd/*.go pkg/*/*.go pkg/*/*/*.go
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -o $@ .
$(BIN): $(LOCALBIN)
	@if [ "$(OS)" = "$(HOSTOS)" -a "$(ARCH)" = "$(HOSTARCH)" ]; then ln -sf $(LOCALBIN) $(BINDIR)/$@; fi
	@if [ "$(OS)" = "$(HOSTOS)" -a "$(ARCH)" = "$(HOSTARCH)" ]; then ln -sf $(LOCALBIN) $@; fi

testbin:
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) build

config: build
	$(LOCALBIN) config add default -v $(DEBUG)

setup: config
	make -C tests DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) setup
	$(LOCALBIN) setup -v $(DEBUG)

run: setup
	$(LOCALBIN) start -v $(DEBUG)

stop: bin
	$(LOCALBIN) stop -v $(DEBUG)

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
	@echo
	@echo "You need install requirements for EVE (look at https://github.com/lf-edge/eve#install-dependencies)."
	@echo "Also, you need to install 'uuidgen' utility."
	@echo "You need access to docker socket and installed qemu packages."

