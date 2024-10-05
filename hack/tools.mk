# This Makefile contains targets for installing various development tools.
# The tools are installed in a local bin directory, making it easy to manage
# project-specific tool versions without affecting the system-wide installation.

################################################################################
# Directory and file path variables                                            #
################################################################################

HACK_DIR       ?= $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
TOOLS_MOD_FILE := $(HACK_DIR)/tools/go.mod
BIN_DIR        ?= $(HACK_DIR)/bin
INCLUDE_DIR    ?= $(HACK_DIR)/include

# Detect OS and architecture
OS   := $(shell uname -s | tr A-Z a-z)
ARCH := $(shell uname -m)

################################################################################
# Tool versions                                                                #
################################################################################

GOLANGCI_LINT_VERSION ?= $(shell grep github.com/golangci/golangci-lint $(TOOLS_MOD_FILE) | awk '{print $$2}')
HELM_VERSION          ?= $(shell grep helm.sh/helm/v3 $(TOOLS_MOD_FILE) | awk '{print $$2}')

################################################################################
# Tool targets                                                                 #
################################################################################

GOLANGCI_LINT  := $(BIN_DIR)/golangci-lint-$(OS)-$(ARCH)-$(GOLANGCI_LINT_VERSION)
HELM           := $(BIN_DIR)/helm-$(OS)-$(ARCH)-$(HELM_VERSION)

$(GOLANGCI_LINT):
	$(call install-golangci-lint,$@,$(GOLANGCI_LINT_VERSION))

$(HELM):
	$(call install-helm,$@,$(HELM_VERSION))

################################################################################
# Symlink targets                                                              #
################################################################################

GOLANGCI_LINT_LINK 	:= $(BIN_DIR)/golangci-lint
HELM_LINK 					:= $(BIN_DIR)/helm

.PHONY: $(GOLANGCI_LINT_LINK)
$(GOLANGCI_LINT_LINK): $(GOLANGCI_LINT)
	$(call create-symlink,$(GOLANGCI_LINT),$(GOLANGCI_LINT_LINK))

.PHONY: $(HELM_LINK)
$(HELM_LINK): $(HELM)
	$(call create-symlink,$(HELM),$(HELM_LINK))

################################################################################
# Alias targets                                                                #
################################################################################

TOOLS := install-golangci-lint install-helm

.PHONY: install-tools
install-tools: $(TOOLS)

.PHONY: install-golangci-lint
install-golangci-lint: $(GOLANGCI_LINT) $(GOLANGCI_LINT_LINK)

.PHONY: install-helm
install-helm: $(HELM) $(HELM_LINK)

################################################################################
# Clean up targets                                                             #
################################################################################

# Clean up all installed tools and symlinks
.PHONY: clean-tools
clean-tools:
	rm -rf $(BIN_DIR)/*
	rm -rf $(INCLUDE_DIR)/*

# Update all tools
.PHONY: update-tools
update-tools: clean-tools install-tools

################################################################################
# Helper functions                                                             #
################################################################################

# install-golangci-lint installs golangci-lint.
#
# $(1) binary path
# $(2) version
define install-golangci-lint
	@[ -f $(1) ] || { \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	echo "Installing golangci-lint $(2) to $(1)" ;\
	curl -fsSL -o install.sh https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh ;\
	chmod 0700 install.sh ;\
	./install.sh -b $$TMP_DIR $(2) ;\
	mkdir -p $(dir $(1)) ;\
	mv $$TMP_DIR/golangci-lint $(1) ;\
	rm -rf $$TMP_DIR ;\
	}
endef

# install-helm installs Helm.
#
# $(1) binary path
# $(2) version
define install-helm
	@[ -f $(1) ] || { \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	echo "Installing helm $(2) to $(1)" ;\
	curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 ;\
	chmod 0700 get_helm.sh ;\
	PATH="$$TMP_DIR:$$PATH" HELM_INSTALL_DIR=$$TMP_DIR USE_SUDO="false" DESIRED_VERSION="$(2)" ./get_helm.sh ;\
	mkdir -p $(dir $(1)) ;\
	mv $$TMP_DIR/helm $(1) ;\
	rm -rf $$TMP_DIR ;\
	}
endef
