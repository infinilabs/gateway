SHELL=/bin/bash

# Basic info
PRODUCT?= gateway
BRANCH?= main
VERSION?= $(shell [[ "$(BRANCH)" == "main" ]] && echo "latest" || echo "$(BRANCH)")
VERSIONS?= "latest"
OUTPUT?= "/tmp/gateway-docs"

# Temporary file path for branches
BRANCH_FILE := $(OUTPUT)/branch_list.txt

.PHONY: build

default: build

config:
	cp docs/config.yaml config.bak
	# Replace placeholder (e.g., "BRANCH") in config.toml with the VERSION environment variable
	sed -i '' "s/BRANCH/$(BRANCH)/g" docs/config.yaml

build: config
	echo $(VERSIONS)
	cd docs && hugo.old --minify --theme book --destination="$(OUTPUT)/$(PRODUCT)/$(VERSION)" \
        --baseURL="/$(PRODUCT)/$(VERSION)" 1> /dev/null
	@$(MAKE) restore-generated-file

restore-generated-file:
	mv config.bak docs/config.yaml