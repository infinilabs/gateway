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
	# Check the OS and set the sed command accordingly
	if [ "$(uname)" == "Darwin" ]; then \
	    sed_command="sed -i .bak"; \
	else \
	    sed_command="sed -i"; \
	fi
	# Debug the sed command
	echo "Using sed command: $(sed_command)"

	# Replace "BRANCH" in config.yaml with the value of VERSION
	$(sed_command) 's/BRANCH/$(VERSION)/g' docs/config.yaml

build: config
	echo $(VERSIONS)
	cd docs && hugo.old --minify --theme book --destination="$(OUTPUT)/$(PRODUCT)/$(VERSION)" \
        --baseURL="/$(PRODUCT)/$(VERSION)" 1> /dev/null
	@$(MAKE) restore-generated-file

restore-generated-file:
	mv config.bak docs/config.yaml