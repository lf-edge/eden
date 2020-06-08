DEBUG ?= "-v debug"

# unless otherwise set, I am building for my own architecture, i.e. not cross-compiling
# and for my OS
ARCH ?= $(HOSTARCH)
OS ?= $(HOSTOS)

WORKDIR ?= $(CURDIR)/../../dist
ECONFIG ?= `$(LOCALBIN) config get`

test: units_test integration_test

units_test:
	make -C units test

integration_test:
	make -C integration DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) ECONFIG=$(ECONFIG) test

build: integration_build

integration_build:
	make -C integration DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) build

setup: integration_setup

integration_setup:
	make -C integration DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) ECONFIG=$(ECONFIG) setup

clean: integration_clean

integration_clean:
	make -C integration DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) ECONFIG=$(ECONFIG) clean
