DIST=$(CURDIR)/dist
BIN=$(DIST)/bin
CONFIG=$(DIST)/Config.in
include $(CONFIG)

HOSTARCH:=$(subst aarch64,arm64,$(subst x86_64,amd64,$(shell uname -m)))

ZARCH ?= $(HOSTARCH)
export ZARCH

clean: stop
	rm -rf $(DIST)/* || sudo rm -rf $(DIST)/*

$(DIST):
	test -d $@ || mkdir -p $@

EVE_DIST=$(DIST)/eve

$(EVE_DIST):
	test -d $@ || mkdir -p $@

ADAM_DIST ?= $(DIST)/adam

$(ADAM_DIST):
	test -d $@ || mkdir -p $@

BIOS_IMG ?= $(EVE_DIST)/dist/$(ZARCH)/OVMF.fd
LIVE_IMG ?= $(EVE_DIST)/dist/$(ZARCH)/live.img 
EVE_URL ?= "https://github.com/lf-edge/eve.git"
EVE_REF ?= "master"
ADAM_URL ?= "https://github.com/lf-edge/adam.git"
ADAM_REF ?= "master"

# any non-empty value will trigger eve rebuild
REBUILD ?=

# any non-empty value will trigger fix eve ip
FIX_IP ?=

ACCEL ?=
SSH_PORT ?= 2222

run: adam_run eve_run

$(CONFIG): save

save: $(DIST)
	echo "# Configuration settings" > $(CONFIG)
	echo ADAM_DIST=$(ADAM_DIST) >> $(CONFIG)
	echo ZARCH=$(ZARCH) >> $(CONFIG)
	echo BIOS_IMG=$(BIOS_IMG) >> $(CONFIG)
	echo LIVE_IMG=$(LIVE_IMG) >> $(CONFIG)
	echo EVE_URL=$(EVE_URL) >> $(CONFIG)
	echo EVE_REF=$(EVE_REF) >> $(CONFIG)
	echo ADAM_URL=$(ADAM_URL) >> $(CONFIG)
	echo ADAM_REF=$(ADAM_REF) >> $(CONFIG)
	echo ACCEL=$(ACCEL) >> $(CONFIG)
	echo SSH_PORT=$(SSH_PORT) >> $(CONFIG)
	echo CERTS_DIST=$(CERTS_DIST) >> $(CONFIG)
	echo DOMAIN=$(DOMAIN) >> $(CONFIG)
	echo IP=$(IP) >> $(CONFIG)
	echo UUID=$(UUID) >> $(CONFIG)
	echo ADAM_PORT=$(ADAM_PORT) >> $(CONFIG)

eve_run: save eve_stop eve_live
	@echo EVE run
	nohup make -C $(EVE_DIST) CONF_DIR=$(ADAM_DIST)/run/config/ SSH_PORT=$(SSH_PORT) ACCEL=$(ACCEL) run >$(DIST)/eve.log 2>&1 & echo "$$!" >$(DIST)/eve.pid
	@echo You can see logs of EVE:
	@echo $(DIST)/eve.log
	@echo You can ssh into EVE:
	@echo ssh -i $(CERTS_DIST)/id_rsa -p $(SSH_PORT) root@127.0.0.1

IMGS := $(LIVE_IMG) $(BIOS_IMG)
IMGS_MISSING := $(shell $(foreach f,$(IMGS),test -e "$f" || echo "$f";))

eve_live: eve_ref
ifneq ($(FIX_IP),)
	chmod a+x $(CURDIR)/scripts/fixIPs.sh
	$(CURDIR)/scripts/fixIPs.sh $(EVE_DIST)/Makefile
endif
ifneq ($(REBUILD),)
	make -C $(EVE_DIST) CONF_DIR=$(ADAM_DIST)/run/config/ live
else
ifneq ($(IMGS_MISSING),)
		make -C $(EVE_DIST) CONF_DIR=$(ADAM_DIST)/run/config/ live
else
		true
endif
endif

eve_ref: eve
	cd $(EVE_DIST); test -n $(EVE_REF) && git checkout -f $(EVE_REF)

eve: $(DIST)
	test -d $(EVE_DIST) || git clone $(EVE_URL) $(EVE_DIST)


CERTS_DIST ?= $(DIST)/certs
DOMAIN ?= mydomain.adam
IP?=$(shell hostname -I | cut -d' ' -f1)
UUID?=$(shell uuidgen)
ADAM_PORT?=3333

adam_docker_stop:
	docker ps|grep eden_adam&&docker stop eden_adam||echo ""
	docker ps --all|grep eden_adam&&docker rm eden_adam||echo ""

adam_run: save adam_docker_stop adam_ref certs_and_config
	@echo Adam Run
	cd $(ADAM_DIST); docker run --name eden_adam -d -v $(ADAM_DIST)/run:/adam/run -p $(ADAM_PORT):8080 lfedge/adam server --conf-dir /tmp
	@echo ADAM is ready for access: https://$(IP):$(ADAM_PORT)

adam_ref: adam
	cd $(ADAM_DIST); test -n $(ADAM_REF) && git checkout -f $(ADAM_REF)

adam:
	test -d $(ADAM_DIST) || git clone $(ADAM_URL) $(ADAM_DIST)

$(CERTS_DIST):
	test -d $@ || mkdir -p $@

certs_and_config: $(CERTS_DIST)
ifeq ($(shell test -f $(ADAM_DIST)/run/adam/server.pem),)
	test -d $(ADAM_DIST)/run/adam || mkdir -p $(ADAM_DIST)/run/adam
	test -d $(ADAM_DIST)/run/config || mkdir -p $(ADAM_DIST)/run/config
	chmod a+x $(CURDIR)/scripts/genCerts.sh
	$(CURDIR)/scripts/genCerts.sh -o $(CERTS_DIST) -i $(IP) -d $(DOMAIN) -u $(UUID)
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

stop: adam_docker_stop eve_stop

eve_stop:
	test -f $(DIST)/eve.pid && kill $(shell cat $(DIST)/eve.pid) && rm $(DIST)/eve.pid || echo ""

test:
	IP=$(IP) ADAM_DIST=$(ADAM_DIST) go test ./tests/integration/adam_test.go -v

$(BIN):
	mkdir -p $(BIN)

bin: elog elogwatch

elog: $(BIN)
	cd cmd/elog/; go build; cp elog $(BIN)

elogwatch: $(BIN)
	cd cmd/elogwatch/; go build; cp elogwatch $(BIN)

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
	@echo "   stop          stop ADAM and EVE"
	@echo "   clean         cleanup directories"
	@echo "   bin		build utilities"
	@echo
	@echo "You need access to docker socket and installed qemu packages"
	@echo "You must set the FIX_IP=true variable if you use subnets 192.168.1.0/24 or 192.168.2.0/24 for any interface on host"

