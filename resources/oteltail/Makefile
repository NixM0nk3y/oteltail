# Meta tasks
# ----------

# Go parameters
GOCMD=GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

export SAM_CLI_TELEMETRY=0

VERSION=1.0

# Useful variables

# region
export AWS_REGION ?= eu-west-1

# Output helpers
# --------------

TASK_DONE = echo "✓  $@ done"
TASK_BUILD = echo "🛠️  $@ done"

export CODE_BUILD_NUMBER ?= 0
export CODE_RESOLVED_SOURCE_VERSION ?=$(shell git rev-list -1 HEAD --abbrev-commit)
export BUILD_DATE=$(shell date -u '+%Y%m%d')

# ----------------
.DEFAULT_GOAL := lambda/build

lambda/build: ## build the lambda
	$(GOBUILD)  -ldflags " \
		-X oteltail/pkg/version.Version=${VERSION}.${CODE_BUILD_NUMBER} \
		-X oteltail/pkg/version.BuildHash=${CODE_RESOLVED_SOURCE_VERSION} \
		-X oteltail/pkg/version.BuildDate=${BUILD_DATE}" -o ./bootstrap -v cmd/oteltail-lambda/*.go
	@$(TASK_BUILD)

lambda/test: lambda/build ## Test a local invoke of the lambda.
	@sam local invoke Lambda --event ./test/eventbridge-s3.json --template test/template.yaml --debug --env-vars test/environment.json  --region $(AWS_REGION)
	@$(TASK_BUILD)

build-Lambda: lambda/build
	cp ./bootstrap ${ARTIFACTS_DIR}/
	@$(TASK_BUILD)

test: 
	@$(GOTEST) -v ./...
	@$(TASK_DONE)

clean: ## clean any generated files
	@$(GOCLEAN)
	@rm -f ./bootstrap
	@$(TASK_BUILD)

help: ## Show this help message.
	@echo 'usage: make [target] ...'
	@echo
	@echo 'targets:'
	@egrep '^(.+)\:\ ##\ (.+)' ${MAKEFILE_LIST} | column -t -c 2 -s ':#'
