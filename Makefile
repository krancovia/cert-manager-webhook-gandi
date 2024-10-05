include $(CURDIR)/hack/tools.mk

SHELL ?= /bin/bash

CERT_MANAGER_CHART_VERSION 	:= 1.14.5

GO_LINT_ERROR_FORMAT 	?= colored-line-number

VERSION_PACKAGE := github.com/krancovia/cert-manager-webhook-gandi/internal/version

IMAGE_REPO 			?= cert-manager-webhook-gandi
IMAGE_TAG 			?= dev
IMAGE_PUSH 			?= false
IMAGE_PLATFORMS 	=

DOCKER_CMD := docker run \
	-it \
	--rm \
	-v gomodcache:/go/pkg/mod \
	-v $(CURDIR):/cert-manager-webhook-gandi \
	-w /cert-manager-webhook-gandi \
	golang:1.23.1-bookworm

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
# Hack: Targets to help you hack                                               #
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

.PHONY: hack-build
hack-build: build-base-image
	docker build --tag cert-manager-webhook-gandi:dev .

.PHONY: hack-install-cert-manager
hack-install-cert-manager: install-helm
	$(HELM) upgrade cert-manager cert-manager \
		--repo https://charts.jetstack.io \
		--version $(CERT_MANAGER_CHART_VERSION) \
		--install \
		--create-namespace \
		--namespace cert-manager \
		--set installCRDs=true \
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
