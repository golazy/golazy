

.DEFAULT_GOAL := help

.PHONY: help
help:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: docs
docs: ## Generates the README.md for all directories
	bash -c 'for a in $$(find -type d | grep -v vscode) ; do cd $$a ; goreadme -recursive -factories -functions > README.md ; cd - ; done'
