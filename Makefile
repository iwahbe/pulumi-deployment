PACK             := deployments
PACKDIR          := sdk
PROJECT          := github.com/iwahbe/pulumi-deployment
NODE_MODULE_NAME := @pulumi/deployments
NUGET_PKG_NAME   := Pulumi.Deployments

PROVIDER        := pulumi-resource-${PACK}
VERSION         ?= 0.1.0
PROVIDER_PATH   := provider
VERSION_PATH    := ${PROVIDER_PATH}/pkg/version.Version

PULUMI          := .pulumi/bin/pulumi

SCHEMA_FILE     := provider/cmd/pulumi-resource-deployments/schema.json
GOPATH          := $(shell go env GOPATH)

WORKING_DIR     := $(shell pwd)
TESTPARALLELISM := 4


schema: $(SCHEMA_FILE)

$(SCHEMA_FILE): provider $(PULUMI)
	$(PULUMI) package get-schema $(WORKING_DIR)/bin/${PROVIDER} | \
		jq 'del(.version)' > $(SCHEMA_FILE)

provider:
	go build -o $(WORKING_DIR)/bin/${PROVIDER} -ldflags "-X ${PROJECT}/${VERSION_PATH}=${VERSION}" $(PROJECT)


# Keep the version of the pulumi binary used for code generation in sync with the version
# of the dependency used by github.com/pulumi/pulumi-deployments/provider

$(PULUMI): HOME := $(WORKING_DIR)
$(PULUMI): go.mod
	@ PULUMI_VERSION="$$(cd provider && go list -m github.com/pulumi/pulumi/pkg/v3 | awk '{print $$2}')"; \
	if [ -x $(PULUMI) ]; then \
		CURRENT_VERSION="$$($(PULUMI) version)"; \
		if [ "$${CURRENT_VERSION}" != "$${PULUMI_VERSION}" ]; then \
			echo "Upgrading $(PULUMI) from $${CURRENT_VERSION} to $${PULUMI_VERSION}"; \
			rm $(PULUMI); \
		fi; \
	fi; \
	if ! [ -x $(PULUMI) ]; then \
		curl -fsSL https://get.pulumi.com | sh -s -- --version "$${PULUMI_VERSION#v}"; \
	fi

.PHONY: provider schema
