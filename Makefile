SHELL := bash

# define standard colors
BLACK        := $(shell tput -Txterm setaf 0)
RED          := $(shell tput -Txterm setaf 1)
GREEN        := $(shell tput -Txterm setaf 2)
YELLOW       := $(shell tput -Txterm setaf 3)
LIGHTPURPLE  := $(shell tput -Txterm setaf 4)
PURPLE       := $(shell tput -Txterm setaf 5)
BLUE         := $(shell tput -Txterm setaf 6)
WHITE        := $(shell tput -Txterm setaf 7)

RESET := $(shell tput -Txterm sgr0)

# add a service here when it uses transifex
L10N_MODULES := \
	services/notifications \
	services/userlog \
	services/graph

# if you add a module here please also add it to the .drone.star file
OCIS_MODULES = \
	services/activitylog \
	services/antivirus \
	services/app-provider \
	services/app-registry \
	services/audit \
	services/auth-basic \
	services/auth-bearer \
	services/auth-machine \
	services/auth-service \
	services/clientlog \
	services/collaboration \
	services/eventhistory \
	services/frontend \
	services/gateway \
	services/graph \
	services/groups \
	services/idm \
	services/idp \
	services/invitations \
	services/nats \
	services/notifications \
	services/ocdav \
	services/ocm \
	services/ocs \
	services/policies \
	services/postprocessing \
	services/proxy \
	services/search \
	services/settings \
	services/sharing \
	services/sse \
	services/storage-system \
	services/storage-publiclink \
	services/storage-shares \
	services/storage-users \
	services/store \
	services/thumbnails \
	services/userlog \
	services/users \
	services/web \
	services/webdav\
	services/webfinger\
	ocis \
	ocis-pkg

# bin file definitions
PHP_CS_FIXER=php -d zend.enable_gc=0 vendor-bin/owncloud-codestyle/vendor/bin/php-cs-fixer
PHP_CODESNIFFER=vendor-bin/php_codesniffer/vendor/bin/phpcs
PHP_CODEBEAUTIFIER=vendor-bin/php_codesniffer/vendor/bin/phpcbf
PHAN=php -d zend.enable_gc=0 vendor-bin/phan/vendor/bin/phan
PHPSTAN=php -d zend.enable_gc=0 vendor-bin/phpstan/vendor/bin/phpstan

ifneq (, $(shell command -v go 2> /dev/null)) # suppress `command not found warnings` for non go targets in CI
include .bingo/Variables.mk
endif

include .make/recursion.mk

.PHONY: help
help:
	@echo "Please use 'make <target>' where <target> is one of the following:"
	@echo
	@echo -e "${GREEN}Testing with test suite natively installed:${RESET}\n"
	@echo -e "${PURPLE}\tdocs: https://owncloud.dev/ocis/development/testing/#testing-with-test-suite-natively-installed${RESET}\n"
	@echo -e "\tmake test-acceptance-api\t\t${BLUE}run API acceptance tests${RESET}"
	@echo -e "\tmake test-acceptance-from-core-api\t\t${BLUE}run core API acceptance tests${RESET}"
	@echo -e "\tmake test-paralleldeployment-api\t${BLUE}run API acceptance tests for parallel deployment${RESET}"
	@echo -e "\tmake clean-tests\t\t\t${BLUE}delete API tests framework dependencies${RESET}"
	@echo
	@echo -e "${BLACK}---------------------------------------------------------${RESET}"
	@echo
	@echo -e "${RED}You also should have a look at other available Makefiles:${RESET}"
	@echo
	@echo -e "${GREEN}oCIS:${RESET}\n"
	@echo -e "${PURPLE}\tdocs: https://owncloud.dev/ocis/development/build/${RESET}\n"
	@echo -e "\tsee ./ocis/Makefile"
	@echo -e "\tor run ${YELLOW}make -C ocis help${RESET}"
	@echo
	@echo -e "${GREEN}Documentation:${RESET}\n"
	@echo -e "${PURPLE}\tdocs: https://owncloud.dev/ocis/development/build-docs/${RESET}\n"
	@echo -e "\tsee ./docs/Makefile"
	@echo -e "\tor run ${YELLOW}make -C docs help${RESET}"
	@echo
	@echo -e "${GREEN}Testing with test suite in docker:${RESET}\n"
	@echo -e "${PURPLE}\tdocs: https://owncloud.dev/ocis/development/testing/#testing-with-test-suite-in-docker${RESET}\n"
	@echo -e "\tsee ./tests/acceptance/docker/Makefile"
	@echo -e "\tor run ${YELLOW}make -C tests/acceptance/docker help${RESET}"
	@echo
	@echo -e "${GREEN}Tools for developing tests:\n${RESET}"
	@echo -e "\tmake test-php-style\t\t${BLUE}run PHP code style checks${RESET}"
	@echo -e "\tmake test-php-style-fix\t\t${BLUE}run PHP code style checks and fix any issues found${RESET}"
	@echo
	@echo -e "${GREEN}Tools for linting gherkin feature files:\n${RESET}"
	@echo -e "\tmake test-gherkin-lint\t\t${BLUE}run lint checks on Gherkin feature files${RESET}"
	@echo -e "\tmake test-gherkin-lint-fix\t${BLUE}apply lint fixes to gherkin feature files${RESET}"
	@echo

.PHONY: clean-tests
clean-tests:
	@rm -Rf vendor-bin/**/vendor vendor-bin/**/composer.lock tests/acceptance/output

BEHAT_BIN=vendor-bin/behat/vendor/bin/behat
# behat config file for parallel deployment tests
PARALLEL_BEHAT_YML=tests/parallelDeployAcceptance/config/behat.yml
# behat config file for core api tests
CORE_BEHAT_YML=tests/acceptance/config/behat-core.yml

.PHONY: test-acceptance-api
test-acceptance-api: vendor-bin/behat/vendor
	BEHAT_BIN=$(BEHAT_BIN) $(PWD)/tests/acceptance/run.sh

.PHONY: test-acceptance-from-core-api
test-acceptance-from-core-api: vendor-bin/behat/vendor
	BEHAT_BIN=$(BEHAT_BIN) BEHAT_YML=$(CORE_BEHAT_YML) $(PWD)/tests/acceptance/run.sh --type core-api

.PHONY: test-paralleldeployment-api
test-paralleldeployment-api: vendor-bin/behat/vendor
	BEHAT_BIN=$(BEHAT_BIN) BEHAT_YML=$(PARALLEL_BEHAT_YML) $(PWD)/tests/acceptance/run.sh

vendor/bamarni/composer-bin-plugin: composer.lock
	composer install

vendor-bin/behat/vendor: vendor/bamarni/composer-bin-plugin vendor-bin/behat/composer.lock
	composer bin behat install --no-progress

vendor-bin/behat/composer.lock: vendor-bin/behat/composer.json
	@echo behat composer.lock is not up to date.
	@rm vendor-bin/behat/composer.lock || true

composer.lock: composer.json
	@echo composer.lock is not up to date.
	@rm composer.lock || true

.PHONY: generate
generate:
	@for mod in $(OCIS_MODULES); do \
        $(MAKE) -C $$mod generate || exit 1; \
    done

.PHONY: vet
vet:
	@for mod in $(OCIS_MODULES); do \
        $(MAKE) --no-print-directory -C $$mod vet  || exit 1; \
    done

.PHONY: clean
clean:
	@for mod in $(OCIS_MODULES); do \
        $(MAKE) --no-print-directory -C $$mod clean || exit 1; \
    done

.PHONY: docs-generate
docs-generate:
	# empty the folders first to only have files that are generated without remnants
	find docs/services/_includes/ -type f \( -name "*" ! -name ".git*" ! -name "_*" \) -delete || exit 1

	@for mod in $(OCIS_MODULES); do \
        $(MAKE) --no-print-directory -C $$mod docs-generate || exit 1; \
    done

	$(MAKE) --no-print-directory -C docs docs-generate || exit 1
	cp docs/services/general-info/env-var-deltas/*.adoc docs/services/_includes/adoc/env-var-deltas/

.PHONY: check-env-var-annotations
check-env-var-annotations:
	.make/check-env-var-annotations.sh

.PHONY: ci-go-generate
ci-go-generate:
	@for mod in $(OCIS_MODULES); do \
        $(MAKE) --no-print-directory -C $$mod ci-go-generate || exit 1; \
    done

.PHONY: ci-node-generate
ci-node-generate:
	@if [ $(MAKE_DEPTH) -le 1 ]; then \
	for mod in $(OCIS_MODULES); do \
        $(MAKE) --no-print-directory -C $$mod ci-node-generate || exit 1; \
    done; fi;

.PHONY: go-mod-tidy
go-mod-tidy:
	@for mod in $(OCIS_MODULES); do \
        $(MAKE) --no-print-directory -C $$mod go-mod-tidy || exit 1; \
    done

.PHONY: test
test:
	@go test -v -tags '$(TAGS)' -coverprofile coverage.out ./...

.PHONY: go-coverage
go-coverage:
	@if [ ! -f coverage.out ]; then $(MAKE) test  &>/dev/null; fi;
	@for mod in $(OCIS_MODULES); do \
        echo -n "% coverage $$mod: "; $(MAKE) --no-print-directory -C $$mod go-coverage || exit 1; \
    done

.PHONY: protobuf
protobuf:
	@for mod in ./services/thumbnails ./services/store ./services/settings; do \
        echo -n "% protobuf $$mod: "; $(MAKE) --no-print-directory -C $$mod protobuf || exit 1; \
    done

.PHONY: golangci-lint
golangci-lint:
	@for mod in $(OCIS_MODULES); do \
        $(MAKE) --no-print-directory -C $$mod golangci-lint; \
    done

.PHONY: ci-golangci-lint
ci-golangci-lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --modules-download-mode vendor --timeout 15m0s --issues-exit-code 0 --out-format checkstyle > checkstyle.xml

.PHONY: golangci-lint-fix
golangci-lint-fix:
	@for mod in $(OCIS_MODULES); do \
        $(MAKE) --no-print-directory -C $$mod golangci-lint-fix; \
    done

.PHONY: test-gherkin-lint
test-gherkin-lint:
	gherlint tests/acceptance/features -c tests/acceptance/config/.gherlintrc.json

.PHONY: test-gherkin-lint-fix
test-gherkin-lint-fix:
	gherlint --fix tests/acceptance/features -c tests/acceptance/config/.gherlintrc.json

.PHONY: bingo-update
bingo-update: $(BINGO)
	$(BINGO) get -l -v

.PHONY: check-licenses
check-licenses: ci-go-check-licenses ci-node-check-licenses

.PHONY: save-licenses
save-licenses: ci-go-save-licenses ci-node-save-licenses

.PHONY: ci-go-check-licenses
ci-go-check-licenses: $(GO_LICENSES)
	$(GO_LICENSES) check ./...

.PHONY: ci-node-check-licenses
ci-node-check-licenses:
	@for mod in $(OCIS_MODULES); do \
        echo -e "% check-license $$mod:"; $(MAKE) --no-print-directory -C $$mod ci-node-check-licenses || exit 1; \
    done

.PHONY: ci-go-save-licenses
ci-go-save-licenses: $(GO_LICENSES)
	@mkdir -p ./third-party-licenses/go/ocis/third-party-licenses
	$(GO_LICENSES) csv ./... > ./third-party-licenses/go/ocis/third-party-licenses.csv
	$(GO_LICENSES) save ./... --force --save_path="./third-party-licenses/go/ocis/third-party-licenses"

.PHONY: ci-node-save-licenses
ci-node-save-licenses:
	@for mod in $(OCIS_MODULES); do \
        $(MAKE) --no-print-directory -C $$mod ci-node-save-licenses || exit 1; \
    done

CHANGELOG_VERSION =

.PHONY: changelog
changelog: $(CALENS)
ifndef CHANGELOG_VERSION
	$(error CHANGELOG_VERSION is undefined)
endif
	mkdir -p ocis/dist
	$(CALENS) --version $(CHANGELOG_VERSION) -o ocis/dist/CHANGELOG.md

.PHONY: govulncheck
govulncheck: $(GOVULNCHECK)
	$(GOVULNCHECK) ./...

.PHONY: l10n-push
l10n-push:
	@for extension in $(L10N_MODULES); do \
		$(MAKE) -C $$extension l10n-push || exit 1; \
	done

.PHONY: l10n-pull
l10n-pull:
	@for extension in $(L10N_MODULES); do \
		$(MAKE) -C $$extension l10n-pull || exit 1; \
	done

.PHONY: l10n-clean
l10n-clean:
	@for extension in $(L10N_MODULES); do \
		$(MAKE) -C $$extension l10n-clean || exit 1; \
	done

.PHONY: l10n-read
l10n-read:
	@for extension in $(L10N_MODULES); do \
		$(MAKE) -C $$extension l10n-read || exit 1; \
    done

.PHONY: l10n-write
l10n-write:
	@for extension in $(L10N_MODULES); do \
		$(MAKE) -C $$extension l10n-write || exit 1; \
    done

.PHONY: ci-format
ci-format: $(BUILDIFIER)
	$(BUILDIFIER) --mode=fix .drone.star

.PHONY: test-php-style
test-php-style: vendor-bin/owncloud-codestyle/vendor vendor-bin/php_codesniffer/vendor
	$(PHP_CS_FIXER) fix -v --diff --allow-risky yes --dry-run
	$(PHP_CODESNIFFER) --cache --runtime-set ignore_warnings_on_exit --standard=phpcs.xml tests/acceptance tests/TestHelpers

.PHONY: test-php-style-fix
test-php-style-fix: vendor-bin/owncloud-codestyle/vendor
	$(PHP_CS_FIXER) fix -v --diff --allow-risky yes
	$(PHP_CODEBEAUTIFIER) --cache --runtime-set ignore_warnings_on_exit --standard=phpcs.xml tests/acceptance

.PHONY: vendor-bin-codestyle
vendor-bin-codestyle: vendor-bin/owncloud-codestyle/vendor

.PHONY: vendor-bin-codesniffer
vendor-bin-codesniffer: vendor-bin/php_codesniffer/vendor

vendor-bin/owncloud-codestyle/vendor: vendor/bamarni/composer-bin-plugin vendor-bin/owncloud-codestyle/composer.lock
	composer bin owncloud-codestyle install --no-progress

vendor-bin/owncloud-codestyle/composer.lock: vendor-bin/owncloud-codestyle/composer.json
	@echo owncloud-codestyle composer.lock is not up to date.

vendor-bin/php_codesniffer/vendor: vendor/bamarni/composer-bin-plugin vendor-bin/php_codesniffer/composer.lock
	composer bin php_codesniffer install --no-progress

vendor-bin/php_codesniffer/composer.lock: vendor-bin/php_codesniffer/composer.json
	@echo php_codesniffer composer.lock is not up to date.
