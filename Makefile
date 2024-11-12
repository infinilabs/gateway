SHELL=/bin/bash

# Basic info
PRODUCT?= gateway
VERSION?= "latest"
VERSIONS?= "latest,v1.3.0"
BRANCH?= "main"
OUTPUT?= "/tmp/gateway-docs"

.PHONY: build

default: build

config:
		cp docs/config.yaml config.bak
		# Replace placeholder (e.g., "BRANCH") in config.toml with the VERSION environment variable
		sed -i '' "s/BRANCH/$(BRANCH)/g" docs/config.yaml

build: config
		cd docs && hugo.old  --minify --theme book  --destination="$(OUTPUT)"/"$(PRODUCT)"/"$(VERSION)"\
                                 		--baseURL="/$(PRODUCT)"/"$(VERSION)" 1> /dev/null
		@$(MAKE) restore-generated-file

restore-generated-file:
		# Restore the original config.toml
		mv config.bak docs/config.yaml