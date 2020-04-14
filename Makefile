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

DIST=$(CURDIR)/dist
BINDIR := $(DIST)/bin
BIN := eden
LOCALBIN := $(BINDIR)/$(BIN)-$(OS)-$(ARCH)
CONFIG=$(DIST)/Config.in
include $(CONFIG)

MEDIA_SIZE=1024M

ZARCH ?= $(HOSTARCH)
export ZARCH

.DEFAULT_GOAL := help

clean: stop
	test -f $(BIN) && rm $(BIN) || echo ""
	rm -rf $(DIST) || sudo rm -rf $(DIST)

$(DIST):
	mkdir -p $@
$(BINDIR):
	mkdir -p $@

EVE_DIST=$(DIST)/eve

IMG_FORMAT ?= qcow2
BASE_IMG_FORMAT ?= img
HV ?= "kvm"
BIOS_IMG ?= $(EVE_DIST)/dist/$(ZARCH)/OVMF.fd
LIVE_IMG ?= $(EVE_DIST)/dist/$(ZARCH)/live.$(IMG_FORMAT)
EVE_URL ?= "https://github.com/lf-edge/eve.git"
EVE_REF ?= 5.1.11
ADAM_URL ?= "https://github.com/lf-edge/adam.git"
ADAM_REF ?= "master"

EVE_SERIAL ?= "31415926"
EVE_REF_OLD=$(EVE_REF)
EVE_DIST_OLD=$(EVE_REF)

EVE_BASE_REF?=5.1.10
EVE_BASE_VERSION?=$(EVE_BASE_REF)-$(ZARCH)
EVE_BASE_DIST=$(DIST)/evebaseos

$(EVE_DIST):
	git clone $(if $(EVE_REF),--branch $(EVE_REF) --single-branch,) $(EVE_URL) $(EVE_DIST)

ADAM_DIST ?= $(DIST)/adam

$(ADAM_DIST):
	git clone $(if $(ADAM_REF),--branch $(ADAM_REF) --single-branch,) $(ADAM_URL) $(ADAM_DIST)

IMAGE_DIST ?= $(DIST)/images

$(IMAGE_DIST):
	mkdir -p $@

BASE_OS_DIST = $(IMAGE_DIST)/baseos

$(BASE_OS_DIST):
	mkdir -p $@

IMAGE_VM_DIST = $(IMAGE_DIST)/vm

$(IMAGE_VM_DIST):
	mkdir -p $@

IMAGE_DOCKER_DIST = $(IMAGE_DIST)/docker

$(IMAGE_DOCKER_DIST):
	mkdir -p $@

# any non-empty value will trigger eve rebuild
REBUILD ?=

# any non-empty value will enable logs checking
LOGS ?=

ACCEL ?= true

SSH_PORT ?= 2222

$(CONFIG): save

ADAM_CA=$(ADAM_DIST)/run/config/root-certificate.pem
EVE_CERT=$(ADAM_DIST)/run/config/onboard.cert.pem

save: $(DIST)
	@echo "# Configuration settings" > $(CONFIG)
	@echo ADAM_DIST=$(ADAM_DIST) >> $(CONFIG)
	@echo ZARCH=$(ZARCH) >> $(CONFIG)
	@echo HV=$(HV) >> $(CONFIG)
	@echo BIOS_IMG=$(BIOS_IMG) >> $(CONFIG)
	@echo LIVE_IMG=$(LIVE_IMG) >> $(CONFIG)
	@echo EVE_URL=$(EVE_URL) >> $(CONFIG)
	@echo EVE_REF=$(EVE_REF) >> $(CONFIG)
	@echo ADAM_URL=$(ADAM_URL) >> $(CONFIG)
	@echo ADAM_REF=$(ADAM_REF) >> $(CONFIG)
	@echo ACCEL=$(ACCEL) >> $(CONFIG)
	@echo SSH_PORT=$(SSH_PORT) >> $(CONFIG)
	@echo CERTS_DIST=$(CERTS_DIST) >> $(CONFIG)
	@echo DOMAIN=$(DOMAIN) >> $(CONFIG)
	@echo IP=$(IP) >> $(CONFIG)
	@echo UUID=$(UUID) >> $(CONFIG)
	@echo ADAM_PORT=$(ADAM_PORT) >> $(CONFIG)
	@echo EVE_SERIAL=$(EVE_SERIAL) >> $(CONFIG)
	@echo EVE_BASE_REF=$(EVE_BASE_REF) >> $(CONFIG)
	@echo EVE_BASE_VERSION=$(EVE_BASE_VERSION) >> $(CONFIG)
	@echo ADAM_CA=$(ADAM_CA) >> $(CONFIG)
	@echo EVE_CERT=$(EVE_CERT) >> $(CONFIG)
	@echo LOGS=$(LOGS) >> $(CONFIG)

QEMU_DTB_PART_amd64=
QEMU_DTB_PART_arm64=--dtb-part=$(EVE_DIST)/dtb
QEMU_DTB_PART=$(QEMU_CONF_OPTS_$(ZARCH))

eve_config: save bin eve_live
	@echo EVE run
	$(LOCALBIN) qemuconf --output=$(DIST)/qemu.conf --image-part=$(LIVE_IMG) --firmware=$(BIOS_IMG) --config-part=$(ADAM_DIST)/run/config --hostfwd="2222=22,5912=5901,5911=5900,8027=8027,8028=8028" --dtb-part=$(QEMU_DTB_PART)

IMGS := $(LIVE_IMG) $(BIOS_IMG)
IMGS_MISSING := $(shell $(foreach f,$(IMGS),test -e "$f" || echo "$f";))

eve_rootfs: $(EVE_DIST)
	make -C $(EVE_DIST) HV=$(HV) CONF_DIR=$(ADAM_DIST)/run/config/ rootfs

eve_live: $(EVE_DIST)
ifneq ($(REBUILD),)
	make -C $(EVE_DIST) HV=$(HV) IMG_FORMAT=$(IMG_FORMAT) CONF_DIR=$(ADAM_DIST)/run/config/ live
	make -C $(EVE_DIST) HV=$(HV) $(BIOS_IMG)
ifneq ($(DEVICETREE_DTB),)
	make -C $(EVE_DIST) HV=$(HV) $(DEVICETREE_DTB)
endif
else
ifneq ($(IMGS_MISSING),)
	make -C $(EVE_DIST) HV=$(HV) IMG_FORMAT=$(IMG_FORMAT) CONF_DIR=$(ADAM_DIST)/run/config/ live
	make -C $(EVE_DIST) HV=$(HV) $(BIOS_IMG)
ifneq ($(DEVICETREE_DTB),)
	make -C $(EVE_DIST) HV=$(HV) $(DEVICETREE_DTB)
endif
else
	true
endif
endif


CERTS_DIST ?= $(DIST)/certs
DOMAIN ?= mydomain.adam

ifeq ($(shell uname -s), Darwin)
IP?=$(shell ifconfig|grep "inet "|grep -Fv 127.0.0.1|awk '{print $$2}'|tail -1)
else
IP?=$(shell hostname -I | cut -d' ' -f1)
endif
UUID?=$(shell uuidgen)
ADAM_PORT?=3333
ESERVER_PORT?=8888

stop: bin
	$(LOCALBIN) stop --adam-rm=true --eserver-pid=$(DIST)/eserver.pid --eve-pid=$(DIST)/eve.pid||echo ""

run: save stop $(ADAM_DIST) certs_and_config build baseos image-bios-alpine image-docker-alpine eve_config
	@echo Adam Run
	$(LOCALBIN) start --adam-dist=$(ADAM_DIST) --adam-port=$(ADAM_PORT) --eserver-port=$(ESERVER_PORT) \
	--eserver-pid=$(DIST)/eserver.pid --eserver-log=$(DIST)/eserver.log --image-dist=$(IMAGE_DIST) \
	--eve-config=$(DIST)/qemu.conf --eve-serial=$(EVE_SERIAL) --eve-accel=$(ACCEL) --eve-pid=$(DIST)/eve.pid \
	--eve-log=$(DIST)/eve.log
	@echo ADAM is ready for access: https://$(IP):$(ADAM_PORT)
	@echo You can see logs of EVE:
	@echo $(DIST)/eve.log
	@echo You can ssh into EVE:
	@echo ssh -i $(CERTS_DIST)/id_rsa -p $(SSH_PORT) root@127.0.0.1

$(CERTS_DIST):
	test -d $@ || mkdir -p $@

certs_and_config: $(CERTS_DIST) bin
ifeq ("$(wildcard $(ADAM_CA))","")
	test -d $(ADAM_DIST)/run/adam || mkdir -p $(ADAM_DIST)/run/adam
	test -d $(ADAM_DIST)/run/config || mkdir -p $(ADAM_DIST)/run/config
	$(LOCALBIN) certs -o $(CERTS_DIST) -i $(IP) -d $(DOMAIN) -u $(UUID)
	cp $(CERTS_DIST)/root-certificate.pem $(ADAM_DIST)/run/config/
	cp $(CERTS_DIST)/onboard.cert.pem $(ADAM_DIST)/run/config/
	cp $(CERTS_DIST)/onboard.key.pem $(ADAM_DIST)/run/config/
	cp $(CERTS_DIST)/server.pem $(ADAM_DIST)/run/adam/
	cp $(CERTS_DIST)/server-key.pem $(ADAM_DIST)/run/adam/
	echo $(IP) $(DOMAIN) >$(ADAM_DIST)/run/config/hosts
	echo $(DOMAIN):$(ADAM_PORT) >$(ADAM_DIST)/run/config/server
	test -f $(CERTS_DIST)/id_rsa.pub || ssh-keygen -t rsa -f $(CERTS_DIST)/id_rsa -q -N ""
	yes | cp -f $(CERTS_DIST)/id_rsa.pub $(ADAM_DIST)/run/config/authorized_keys
endif

eve_clean: stop
	rm -rf $(LIVE_IMG)
	rm -rf $(ADAM_DIST)/run/adam/device/*|| sudo rm -rf $(ADAM_DIST)/run/adam/device/*

test_controller:
	LOGS=$(LOGS) ADAM_IP=$(IP) ADAM_DIST=$(ADAM_DIST) EVE_BASE_REF=$(EVE_BASE_REF) ZARCH=$(ZARCH) ADAM_PORT=$(ADAM_PORT) ADAM_CA=$(ADAM_CA) EVE_CERT=$(EVE_CERT) SSH_KEY=$(CERTS_DIST)/id_rsa.pub EVE_SERIAL=$(EVE_SERIAL) go test ./tests/integration/controller_test.go ./tests/integration/common.go -v -count=1 -timeout 3000s

test_base_image: test_controller
	LOGS=$(LOGS) HV=$(HV) ADAM_IP=$(IP) ADAM_DIST=$(ADAM_DIST) EVE_BASE_REF=$(EVE_BASE_REF) ZARCH=$(ZARCH) ADAM_PORT=$(ADAM_PORT) ADAM_CA=$(ADAM_CA) EVE_CERT=$(EVE_CERT) SSH_KEY=$(CERTS_DIST)/id_rsa.pub go test ./tests/integration/baseImage_test.go ./tests/integration/common.go -v -count=1 -timeout 4500s

test_network_instance: test_controller
	LOGS=$(LOGS) ADAM_IP=$(IP) ADAM_DIST=$(ADAM_DIST) EVE_BASE_REF=$(EVE_BASE_REF) ZARCH=$(ZARCH) ADAM_PORT=$(ADAM_PORT) ADAM_CA=$(ADAM_CA) EVE_CERT=$(EVE_CERT) SSH_KEY=$(CERTS_DIST)/id_rsa.pub go test ./tests/integration/networkInstance_test.go ./tests/integration/common.go -v -count=1 -timeout 4000s

test_application_instance: test_controller test_network_instance
	LOGS=$(LOGS) ADAM_IP=$(IP) ADAM_DIST=$(ADAM_DIST) EVE_BASE_REF=$(EVE_BASE_REF) ZARCH=$(ZARCH) ADAM_PORT=$(ADAM_PORT) ADAM_CA=$(ADAM_CA) EVE_CERT=$(EVE_CERT) SSH_KEY=$(CERTS_DIST)/id_rsa.pub go test ./tests/integration/application_test.go ./tests/integration/common.go -v -count=1 -timeout 4000s

test: test_base_image test_network_instance test_application_instance

bin: $(BIN)
build: $(BIN)
$(LOCALBIN): $(BINDIR) cmd/*.go pkg/*/*.go pkg/*/*/*.go
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build -o $@ .
$(BIN): $(LOCALBIN)
	@if [ "$(OS)" = "$(HOSTOS)" -a "$(ARCH)" = "$(HOSTARCH)" -a ! -e "$@" ]; then ln -s $(LOCALBIN) $@; fi

SHA256_CMD = sha256sum
ifeq ($(shell uname -s), Darwin)
        SHA256_CMD = openssl sha256 -r
endif

BASEOSFILE=$(IMAGE_DIST)/baseos/baseos.qcow2

baseos: save $(BASE_OS_DIST) certs_and_config $(BASEOSFILE)

.PRECIOUS: $(BASEOSFILE)

$(BASEOSFILE):
	$(MAKE) eve_rootfs EVE_REF=$(EVE_BASE_REF) EVE_DIST=$(EVE_BASE_DIST) IMG_FORMAT=$(BASE_IMG_FORMAT)
	cp $(EVE_BASE_DIST)/dist/$(ZARCH)/installer/rootfs.$(BASE_IMG_FORMAT) $(BASEOSFILE)
	cd $(IMAGE_DIST)/baseos; $(SHA256_CMD) baseos.qcow2>baseos.sha256
	echo EVE_VERSION>$(IMAGE_DIST)/version.yml.in
	$(MAKE) -C $(EVE_BASE_DIST) $(IMAGE_DIST)/version.yml
	$(MAKE) save_ref_dist_base EVE_REF=$(EVE_REF_OLD) EVE_DIST=$(EVE_DIST_OLD)
	rm -rf $(IMAGE_DIST)/version.yml.in
	rm -rf $(IMAGE_DIST)/version.yml

image-efi-%: baseos bin $(IMAGE_VM_DIST) $(IMAGE_VM_DIST)/%-efi.qcow2
	echo image-efi-$*

.PRECIOUS: $(IMAGE_VM_DIST)/%-efi.qcow2
$(IMAGE_VM_DIST)/%-efi.qcow2:
	$(MAKE) -C $(EVE_BASE_DIST) $(CURDIR)/images/vm/$*/$*.yml
	cd $(IMAGE_VM_DIST) && PATH="$(EVE_BASE_DIST)/build-tools/bin:$(PATH)" linuxkit build -format qcow2-efi -dir $(IMAGE_VM_DIST) -size $(MEDIA_SIZE) $(CURDIR)/images/vm/$*/$*.yml
	cd $(IMAGE_VM_DIST) && $(SHA256_CMD) $*-efi.qcow2>$*-efi.sha256

image-bios-%: baseos bin $(IMAGE_VM_DIST) $(IMAGE_VM_DIST)/%.qcow2
	echo image-bios-$*

.PRECIOUS: $(IMAGE_VM_DIST)/%.qcow2
$(IMAGE_VM_DIST)/%.qcow2:
	$(MAKE) -C $(EVE_BASE_DIST) $(CURDIR)/images/vm/$*/$*.yml
	cd $(IMAGE_VM_DIST) && PATH="$(EVE_BASE_DIST)/build-tools/bin:$(PATH)" linuxkit build -format qcow2-bios -dir $(IMAGE_VM_DIST) -size $(MEDIA_SIZE) $(CURDIR)/images/vm/$*/$*.yml
	cd $(IMAGE_VM_DIST) && rm -rf $(IMAGE_DIST)/vm/linuxkit && $(SHA256_CMD) $*.qcow2>$*.sha256
	@rm $(CURDIR)/images/vm/$*/$*.yml

image-docker-%: baseos bin $(IMAGE_DOCKER_DIST) $(IMAGE_DOCKER_DIST)/%.tar
	echo image-docker-$*

.PRECIOUS: $(IMAGE_DOCKER_DIST)/%.tar
$(IMAGE_DOCKER_DIST)/%.tar:
	$(MAKE) -C $(EVE_BASE_DIST) $(CURDIR)/images/docker/$*/$*.yml
	docker build $(CURDIR)/images/docker/$* -f $(CURDIR)/images/docker/$*/$*.yml -t local-$*
	$(LOCALBIN) ociimage -i local-$* -o $(IMAGE_DOCKER_DIST)/$*.tar -l
	@rm $(CURDIR)/images/docker/$*/$*.yml

save_ref_dist_base:
	$(eval EVE_BASE_VERSION := $(shell cat $(IMAGE_DIST)/version.yml))
	$(MAKE) save EVE_REF=$(EVE_REF_OLD) EVE_DIST=$(EVE_DIST_OLD) EVE_BASE_VERSION=$(EVE_BASE_VERSION)

show-config:
	cat $(CONFIG)

help:
	@echo "EDEN is the harness for testing EVE and ADAM"
	@echo
	@echo "This Makefile automates commons tasks of building and running"
	@echo "  * EVE"
	@echo "  * ADAM"
	@echo "You can set Git repository by EVE_URL/ADAM_URL and tag/branch by EVE_REF/ADAM_REF/EVE_BASE_REF variables."
	@echo
	@echo "Commonly used maintenance and development targets:"
	@echo "   run           run ADAM and EVE"
	@echo "   test          run tests"
	@echo "   stop          stop ADAM and EVE"
	@echo "   clean         full cleanup of test harness"
	@echo "   eve-clean     cleanup of EVE instance related things"
	@echo "   show-config   displays current configuration settings"
	@echo "   build         build utilities (OS and ARCH options supported, for ex. OS=linux ARCH=arm64)"
	@echo
	@echo "You need install requirements for EVE (look at https://github.com/lf-edge/eve#install-dependencies)."
	@echo "Also, you need to install 'uuidgen' utility."
	@echo "You need access to docker socket and installed qemu packages."
	@echo "If you have troubles running EVE, try setting ACCEL=''"
	@echo "The SSH port for accessing the EVE instance can be set by the SSH_PORT variable."
