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

WORKDIR ?= $(CURDIR)/../dist

test: units_test integration_test

build: integration_build reboot_build
setup: integration_setup reboot_setup
clean: integration_clean reboot_clean

units_test:
	make -C units test

integration_test: integration_build integration_setup
	make -C integration DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) test

reboot_test: reboot_build reboot_setup
	make -C reboot DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) test

integration_build:
	make -C integration DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) build

reboot_build:
	make -C reboot DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) build

integration_setup:
	make -C integration DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) setup

reboot_setup:
	make -C reboot DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) setup

integration_clean:
	make -C integration DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) clean

reboot_clean:
	make -C reboot DEBUG=$(DEBUG) ARCH=$(ARCH) OS=$(OS) WORKDIR=$(WORKDIR) clean
