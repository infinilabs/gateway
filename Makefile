SHELL=/bin/bash

# APP info
APP_NAME := gateway
APP_VERSION := 1.0.0_SNAPSHOT
APP_CONFIG := $(APP_NAME).yml
APP_EOLDate := "2027-12-31T10:10:10Z"
APP_STATIC_FOLDER := .public
APP_STATIC_PACKAGE := public
APP_UI_FOLDER := ui
APP_PLUGIN_FOLDER := pipeline proxy
GOMODULE := true

include ../framework/Makefile

CI_ROOT_IMPORT := infini.sh/$(APP_NAME)
GOLANGCI_LINT_FLAGS ?=

.PHONY: ci-test ci-lint

ci-test: config
	@packages="$$( $(GO) list ./... 2>/dev/null | grep -v '^$(CI_ROOT_IMPORT)$$' || true )"; \
	if [[ -z "$$packages" ]]; then echo "no test packages found"; exit 1; fi; \
	$(GOTEST) -v $(GOFLAGS) -timeout 30m $$packages
	@$(MAKE) restore-generated-file

ci-lint: config
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v2.1.6; \
	fi
	@dirs="$$( $(GO) list -f '{{if ne .ImportPath "$(CI_ROOT_IMPORT)"}}{{.Dir}}{{end}}' ./... 2>/dev/null | sed '/^$$/d' | sed 's#^$(CURDIR)/#./#' || true )"; \
	if [[ -z "$$dirs" ]]; then echo "no lint packages found"; exit 1; fi; \
	$(if $(filter $(GOMODULE),true),golangci-lint run $(GOLANGCI_LINT_FLAGS) $$dirs,GO15VENDOREXPERIMENT="1" GO111MODULE=off golangci-lint run $(GOLANGCI_LINT_FLAGS) $$dirs)
	@$(MAKE) restore-generated-file
