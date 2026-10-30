SHELL := /bin/bash
MISE := MISE_CONFIG_DIR="$(CURDIR)/.config/mise" mise

.PHONY: help install install-apply tasks lint test check versions go-fmt go-lint go-test

help:
	@printf '%s\n' \
		'Targets:' \
		'  install         Preview tool and dependency installation' \
		'  install-apply   Install mise-managed tools and project dependencies' \
		'  tasks           List mise tasks' \
		'  lint            Check Go formatting and run go vet' \
		'  test            Run all Go tests' \
		'  check           Run Go formatting checks, go vet, and tests' \
		'  go-fmt          Format Go source files' \
		'  go-lint         Check Go formatting and run go vet' \
		'  go-test         Run all Go tests' \
		'  versions        Print development tool versions'

install:
	MISE_TRUSTED_CONFIG_PATHS="$(CURDIR)" $(MISE) install --locked --dry-run
	MISE_TRUSTED_CONFIG_PATHS="$(CURDIR)" $(MISE) run --dry-run --skip-tools install

install-apply:
	$(MISE) trust --yes "$(CURDIR)/.config/mise/config.toml"
	$(MISE) install --locked
	$(MISE) run install

tasks:
	$(MISE) tasks

lint:
	$(MISE) run lint

test:
	$(MISE) run test

check:
	$(MISE) run check

versions:
	$(MISE) run versions

go-fmt:
	$(MISE) run go-fmt

go-lint:
	$(MISE) run go-lint

go-test:
	$(MISE) run go-test
