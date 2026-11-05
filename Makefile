GOLANGCI_LINT_VERSION := v2.15.0
GO_ARCH_LINT_VERSION := v1.19.1

BIN := $(CURDIR)/bin
GOLANGCI_LINT := $(BIN)/golangci-lint
GO_ARCH_LINT := $(BIN)/go-arch-lint
AGENT := runtimes/jvm/agent
AGENT_JAR := internal/runtime/jvm/komar-agent.jar
AGENT_SOURCES := $(shell find $(AGENT)/src/main $(AGENT)/build.gradle.kts -type f 2>/dev/null)
HAS_GO := $(shell git ls-files --cached --others --exclude-standard -- '*.go' | head -1)

.PHONY: build lint check test lint-go lint-arch lint-java lint-comments agent

build: $(AGENT_JAR)
	go build -o $(BIN)/komar ./cmd/komar

agent: $(AGENT_JAR)

$(AGENT_JAR): $(AGENT_SOURCES)
	cd $(AGENT) && ./gradlew --quiet shadowJar
	cp $(AGENT)/build/libs/komar-agent.jar $@

check: lint test

lint: lint-comments lint-go lint-arch lint-java

lint-comments:
	@scripts/no-comments.sh

lint-go: $(GOLANGCI_LINT) $(AGENT_JAR)
	$(if $(HAS_GO),$(GOLANGCI_LINT) run ./...,@echo 'lint-go: no Go files')

lint-arch: $(GO_ARCH_LINT) $(AGENT_JAR)
	$(if $(HAS_GO),$(GO_ARCH_LINT) check,@echo 'lint-arch: no Go files')

lint-java:
	cd $(AGENT) && ./gradlew --quiet checkstyleMain compileJava compileTestJava test --tests '*ArchitectureTest'

test: $(AGENT_JAR)
	$(if $(HAS_GO),go test ./...,@echo 'test: no Go files')
	cd $(AGENT) && ./gradlew --quiet test

$(GOLANGCI_LINT):
	GOBIN=$(BIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(GO_ARCH_LINT):
	GOBIN=$(BIN) go install github.com/fe3dback/go-arch-lint@$(GO_ARCH_LINT_VERSION)
