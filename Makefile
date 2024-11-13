SHELL=/bin/bash

# Basic info
PRODUCT?= gateway
BRANCH?= main
VERSION?= $(shell [[ "$(BRANCH)" == "main" ]] && echo "latest" || echo "$(BRANCH)")
VERSIONS?= "latest"
OUTPUT?= "/tmp/gateway-docs"

.PHONY: build

default: build


config:
	cp docs/config.yaml config.bak
	# Detect OS and apply the appropriate sed command
	@if [ "$$(uname)" = "Darwin" ]; then \
		echo "Running on macOS"; \
		sed -i '' "s/BRANCH/$(VERSION)/g" docs/config.yaml; \
	else \
		echo "Running on Linux"; \
		sed -i 's/BRANCH/$(VERSION)/g' docs/config.yaml; \
	fi

build: config
	echo $(VERSIONS)
	cd docs && hugo --minify --theme book --destination="$(OUTPUT)/$(PRODUCT)/$(VERSION)" \
        --baseURL="/$(PRODUCT)/$(VERSION)" 1> /dev/null
	@$(MAKE) restore-generated-file

restore-generated-file:
	mv config.bak docs/config.yaml
