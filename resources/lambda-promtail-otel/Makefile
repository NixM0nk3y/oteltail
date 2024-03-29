# Meta tasks
# ----------

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

VERSION=1.0

# Useful variables

# region
export AWS_REGION ?= eu-west-1

# Output helpers
# --------------

TASK_DONE = echo "✓  $@ done"
TASK_BUILD = echo "🛠️  $@ done"

export CODEBUILD_BUILD_NUMBER ?= 0
export CODEBUILD_RESOLVED_SOURCE_VERSION ?=$(shell git rev-list -1 HEAD --abbrev-commit)
export BUILD_DATE=$(shell date -u '+%Y%m%d')

# ----------------
.DEFAULT_GOAL := help

lambda/build: ## build the lambda
	go build -o ./bootstrap lambda-oteltail/*.go
	@$(TASK_BUILD)

lambda/test: lambda/build ## Test a local invoke of the lambda.
	@sam local invoke Lambda --event test/kinesis-cwevent.json --template test/template.yaml --env-vars test/environment.json  --region $(AWS_REGION)
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
