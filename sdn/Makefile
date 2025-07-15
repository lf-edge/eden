DOCKER_TARGET ?= build
DOCKER_PLATFORM ?= $(shell uname -s | tr '[A-Z]' '[a-z]')/$(subst aarch64,arm64,$(subst x86_64,amd64,$(shell uname -m)))
DOCKER_ARCH := $(word 2,$(subst /, ,$(DOCKER_PLATFORM)))
LINUXKIT ?= linuxkit

SDN_REPO ?= "lfedge/eden-sdn"
SDN_DIR=$(CURDIR)
SDN_VM_DIR=$(SDN_DIR)/vm
SDN_SERVICE_CONTAINER := $(shell $(LINUXKIT) pkg show-tag $(SDN_VM_DIR))-$(DOCKER_ARCH)
SDN_VERSION := $(shell grep -v '^#' VERSION | head -n1)

build:
	@$(LINUXKIT) pkg build --platforms $(DOCKER_PLATFORM) --build-yml build.yml $(SDN_VM_DIR)
	@sed 's|SDN_SERVICE_CONTAINER|$(SDN_SERVICE_CONTAINER)|g' $(SDN_VM_DIR)/sdn-vm.yml.in | \
		$(LINUXKIT) build --docker --arch $(DOCKER_ARCH) --name sdn --format raw-bios --dir $(SDN_VM_DIR) -
	@docker buildx build --$(DOCKER_TARGET) --platform $(DOCKER_PLATFORM) \
		-f $(SDN_VM_DIR)/Dockerfile.vm --build-arg SDN_VERSION=$(SDN_VERSION) \
		--tag $(SDN_REPO):$(SDN_VERSION) $(SDN_VM_DIR)