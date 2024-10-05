include $(CURDIR)/hack/tools.mk

SHELL ?= /bin/bash

CERT_MANAGER_CHART_VERSION 	:= 1.16.2

GO_LINT_ERROR_FORMAT 	?= colored-line-number

VERSION_PACKAGE := github.com/krancovia/cert-manager-webhook-gandi/internal/version

IMAGE_REPO 			?= cert-manager-webhook-gandi
IMAGE_TAG 			?= dev
IMAGE_PUSH 			?= false

DOCKER_CMD := docker run \
	-it \
	--rm \
	-v gomodcache:/go/pkg/mod \
	-v $(CURDIR):/cert-manager-webhook-gandi \
	-w /cert-manager-webhook-gandi \
	golang:1.23.4-bookworm

NODE_DOCKER_CMD := docker run \
	-it \
	--rm \
	-v nodemodcache:/usr/local/lib/node_modules \
	-v $(CURDIR):/cert-manager-webhook-gandi \
	-w /cert-manager-webhook-gandi \
	node:22.12.0-bookworm

################################################################################
# Tests                                                                        #
################################################################################

.PHONY: lint-go
lint-go: install-golangci-lint
	$(GOLANGCI_LINT) run --out-format=$(GO_LINT_ERROR_FORMAT)

.PHONY: lint-chart
lint-chart: install-helm
	cd chart && \
	$(HELM) dep up && \
	$(HELM) lint .

.PHONY: test-unit
test-unit:
	TEST_ZONE_NAME=krancovia.io. \
	go test \
		-v \
		-timeout=300s \
		-race \
		-count=1 \
		./...

.PHONY: test
test:
	TEST_ZONE_NAME=krancovia.io. \
	go test \
		-v \
		-timeout=1200s \
		-race \
		-count=1 \
		-coverprofile=coverage.txt \
		-covermode=atomic \
		-tags=integration \
		./...

################################################################################
# Doc generation                                                               #
################################################################################

.PHONY:
docgen:
	hack/chart-docs/docgen.sh

################################################################################
# Hack: Targets to help you hack                                               #
#                                                                              #
# Many of these targets are executed within a Docker container to ensure a     #
# consistent environment that's adequately similar to CI. Any such targets     #
# can be run natively on the host by dropping the `hack-` prefix.              #
################################################################################

.PHONY: hack-lint-go
hack-lint-go:
	$(DOCKER_CMD) make lint-go

.PHONY: hack-lint-chart
hack-lint-chart:
	$(DOCKER_CMD) make lint-chart

.PHONY: hack-test-unit
hack-test-unit:
	$(DOCKER_CMD) make test-unit

.PHONY: hack-test
hack-test:
	$(DOCKER_CMD) make test

.PHONY:
hack-docgen:
	$(NODE_DOCKER_CMD) hack/chart-docs/docgen.sh

.PHONY: hack-build
hack-build:
	docker build --tag cert-manager-webhook-gandi:dev .

.PHONY: hack-install-cert-manager
hack-install-cert-manager: install-helm
	$(HELM) upgrade cert-manager cert-manager \
		--repo https://charts.jetstack.io \
		--version $(CERT_MANAGER_CHART_VERSION) \
		--install \
		--create-namespace \
		--namespace cert-manager \
		--set crds.enabled=true \
		--wait

.PHONY: hack-uninstall-cert-manager
hack-uninstall-cert-manager: install-helm
	$(HELM) delete cert-manager --namespace cert-manager

.PHONY: hack-deploy-issuer-example
hack-deploy-issuer-example:
	@if [ -z "$$EMAIL" ]; then echo "Error: EMAIL is not set"; exit 1; fi
	@if [ -z "$$ZONE" ]; then echo "Error: ZONE is not set"; exit 1; fi
	@if [ -z "$$TOKEN" ]; then echo "Error: TOKEN is not set"; exit 1; fi
	cat examples/issuer.yaml | envsubst '$$EMAIL $$ZONE $$TOKEN' | kubectl apply -f -

.PHONY: hack-clean-issuer-example
hack-clean-issuer-example:
	kubectl delete -f examples/issuer.yaml

.PHONY: hack-deploy-cluster-issuer-example
hack-deploy-cluster-issuer-example:
	@if [ -z "$$EMAIL" ]; then echo "Error: EMAIL is not set"; exit 1; fi
	@if [ -z "$$ZONE" ]; then echo "Error: ZONE is not set"; exit 1; fi
	@if [ -z "$$TOKEN" ]; then echo "Error: TOKEN is not set"; exit 1; fi
	cat examples/cluster-issuer.yaml | envsubst '$$EMAIL $$ZONE $$TOKEN' | kubectl apply -f -

.PHONY: hack-clean-cluster-issuer-example
hack-clean-cluster-issuer-example:
	kubectl delete -f examples/cluster-issuer.yaml
