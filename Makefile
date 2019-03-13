# Setup some useful vars
PKG = github.com/mesosphere/installer
BUILD_OUTPUT = out-$(GOOS)

# Set the build version
ifeq ($(origin VERSION), undefined)
	VERSION := $(shell git describe --tags --always --dirty)
endif
# Set the build branch
ifeq ($(origin BRANCH), undefined)
	BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
endif
# build date
ifeq ($(origin BUILD_DATE), undefined)
	BUILD_DATE := $(shell date -u)
endif
# If no target is defined, assume the host is the target.
ifeq ($(origin GOOS), undefined)
	GOOS := $(shell go env GOOS)
endif
# Lots of these target goarches probably won't work,
# since we depend on vendored packages also being built for the correct arch
ifeq ($(origin GOARCH), undefined)
	GOARCH := $(shell go env GOARCH)
endif
# If no target is defined, assume the host is the target.
ifeq ($(origin HOST_GOOS), undefined)
	HOST_GOOS := $(shell go env GOOS)
endif
# Lots of these target goarches probably won't work,
# since we depend on vendored packages also being built for the correct arch
ifeq ($(origin HOST_GOARCH), undefined)
	HOST_GOARCH := $(shell go env GOARCH)
endif
# Used by the integration tests to tag nodes
ifeq ($(origin CREATED_BY), undefined)
	CREATED_BY := $(shell hostname)
endif

# Versions of external dependencies
ANSIBLE_VERSION = 2.3.0.0
GO_VERSION = 1.11.4
KUBECTL_VERSION = v1.13.4

install: 
	@echo Building in container
	@docker run                                \
	    --rm                                   \
	    -e GOOS="$(GOOS)"                      \
	    -e HOST_GOOS="linux"                   \
	    -e VERSION="$(VERSION)"                \
	    -e BUILD_DATE="$(BUILD_DATE)"          \
	    -u root:root                           \
	    -v "$(shell pwd)":"/src/$(PKG)"        \
	    -w /src/$(PKG)                         \
	    installer-base                         \
	    make bin/$(GOOS)/installer copy-installer copy-playbooks

dist: shallow-clean
	@echo "Running dist inside contianer"
	@docker run                                \
	    --rm                                   \
	    -e GOOS="$(GOOS)"                      \
	    -e HOST_GOOS="linux"                   \
	    -e VERSION="$(VERSION)"                \
	    -e BUILD_DATE="$(BUILD_DATE)"          \
	    -u root:root                           \
	    -v "$(shell pwd)":"/src/$(PKG)"        \
	    -w "/src/$(PKG)"                       \
	    installer-base                          \
	    make dist-common

clean: 
	rm -rf bin
	rm -rf out-*
	rm -rf vendor
	rm -rf vendor-*
	rm -rf tmp

test:
	@docker run                             \
	    --rm                                \
	    -e HOST_GOOS="linux"                \
	    -u root:root                        \
	    -v "$(shell pwd)":/src/$(PKG)       \
	    -w /src/$(PKG)                      \
	    installer-base                      \
	    make test-host

.PHONY: builder
builder:
	docker build                                \
	    --target builder_base                   \
	    -f build/docker/Dockerfile -t installer-base .

# YOU SHOULDN'T NEED TO USE ANYTHING BENEATH THIS LINE
# UNLESS YOU REALLY KNOW WHAT YOU'RE DOING
# ---------------------------------------------------------------------
all:
	@$(MAKE) GOOS=darwin dist
	@$(MAKE) GOOS=linux dist

shallow-clean:
	rm -rf $(BUILD_OUTPUT)

tar-clean: 
	rm installer-*.tar.gz

build: 
	@echo Building installer in container
	@docker run                                \
	    --rm                                   \
	    -e GOOS="$(GOOS)"                      \
	    -e HOST_GOOS="linux"                   \
	    -e VERSION="$(VERSION)"                \
	    -e BUILD_DATE="$(BUILD_DATE)"          \
	    -u root:root                           \
	    -v "$(shell pwd)":"/src/$(PKG)"        \
	    -w /src/$(PKG)                         \
	    installer-base                         \
	    make build-host

copy-all: copy-vendors copy-playbooks copy-installer

copy-installer:
	mkdir -p $(BUILD_OUTPUT)
	cp bin/$(GOOS)/installer $(BUILD_OUTPUT)

copy-playbooks:
	mkdir -p $(BUILD_OUTPUT)/ansible/playbooks
	cp -r $(wildcard ansible/*) $(BUILD_OUTPUT)/ansible/playbooks

copy-vendors: #
	mkdir -p $(BUILD_OUTPUT)/ansible
	cp -r vendor-ansible/out/ansible/* $(BUILD_OUTPUT)/ansible
	cp vendor-kubectl/out/kubectl-$(KUBECTL_VERSION)-$(GOOS)-$(GOARCH) $(BUILD_OUTPUT)/kubectl

tarball: 
	rm -f installer-$(GOOS).tar.gz
	tar -czf installer-$(GOOS).tar.gz -C $(BUILD_OUTPUT) .

# RECIPES BELOW THIS LINE ARE INTENDED FOR CI ONLY. RUN LOCALLY AT YOUR OWN RISK.
# ---------------------------------------------------------------------

all-host:
	@$(MAKE) GOOS=darwin dist-host
	@$(MAKE) GOOS=linux dist-host

test-host:
	go test ./cmd/... ./pkg/... $(TEST_OPTS)

build-host: bin/$(GOOS)/installer

.PHONY: bin/$(GOOS)/installer
bin/$(GOOS)/installer:
	go build -o $@                                                              \
	    -ldflags "-X main.version=$(VERSION) -X 'main.buildDate=$(BUILD_DATE)'" \
	    ./cmd/cli/main.go

vendor: vendor-ansible/out/ansible.tar.gz vendor-kubectl/out/kubectl-$(KUBECTL_VERSION)-$(GOOS)-$(GOARCH)

vendor-ansible/out/ansible.tar.gz:
	mkdir -p vendor-ansible/out
	# TODO don't depend on this
	curl -L https://github.com/apprenda/vendor-ansible/releases/download/v$(ANSIBLE_VERSION)/ansible.tar.gz -o vendor-ansible/out/ansible.tar.gz
	tar -zxf vendor-ansible/out/ansible.tar.gz -C vendor-ansible/out
	rm vendor-ansible/out/ansible.tar.gz

vendor-kubectl/out/kubectl-$(KUBECTL_VERSION)-$(GOOS)-$(GOARCH):
	mkdir -p vendor-kubectl/out/
	curl -L https://storage.googleapis.com/kubernetes-release/release/$(KUBECTL_VERSION)/bin/$(GOOS)/$(GOARCH)/kubectl -o vendor-kubectl/out/kubectl-$(KUBECTL_VERSION)-$(GOOS)-$(GOARCH)
	chmod +x vendor-kubectl/out/kubectl-$(KUBECTL_VERSION)-$(GOOS)-$(GOARCH)

dist-common: vendor build-host copy-all

dist-host: shallow-clean dist-common

get-ginkgo:
	go get github.com/onsi/ginkgo/ginkgo
	cd integration-tests

docs/generate-installer-cli:
	mkdir -p docs/installer-cli
	go run cmd/installer-docs/main.go
	cp docs/installer-cli/installer.md docs/installer-cli/README.md
