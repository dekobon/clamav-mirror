PACKAGE_ROOT = github.com/dekobon
PACKAGE      = $(PACKAGE_ROOT)/clamav-mirror
DATE        ?= $(shell date -u +%FT%T%z)
VERSION     ?= $(shell cat $(CURDIR)/.version 2> /dev/null || echo unknown)
GITHASH     ?= $(shell git rev-parse HEAD)

GOPATH       = $(CURDIR)/.gopath
BIN          = $(GOPATH)/bin
BASE         = $(GOPATH)/src/$(PACKAGE)
PKGS         = $(or $(PKG),$(shell cd $(BASE) && env GOPATH=$(GOPATH) $(GO) list ./... | grep -v "^$(PACKAGE)/vendor/"))
TESTPKGS     = $(shell env GOPATH=$(GOPATH) $(GO) list -f '{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' $(PKGS))
ARCH         = $(shell uname -m | sed -e 's/x86_64/amd64/g' -e 's/i686/i386/g')
PLATFORM     = $(shell uname | tr '[:upper:]' '[:lower:]')

GO      = go
GODOC   = godoc
GOFMT   = gofmt
GLIDE   = glide
TIMEOUT = 15
V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m▶\033[0m")

export GOPATH:=$(GOPATH)

.PHONY: all

all: fmt lint sigupdate sigserver

$(BASE):
	$(info $(M) setting GOPATH…)
	@mkdir -p $(GOPATH)/src/github.com
	@ln -sf $(CURDIR)/src/$(PACKAGE_ROOT) $(GOPATH)/src/$(PACKAGE_ROOT)

# Tools

GOLINT = $(BIN)/golint
$(BIN)/golint:
	$(info $(M) building golint…)
	$Q go get github.com/golang/lint/golint

GOCOVMERGE = $(BIN)/gocovmerge
$(BIN)/gocovmerge:
	$(info $(M) building gocovmerge…)
	$Q go get github.com/wadey/gocovmerge

GOCOV = $(BIN)/gocov
$(BIN)/gocov:
	$(info $(M) building gocov…)
	$Q go get github.com/axw/gocov/...

GOCOVXML = $(BIN)/gocov-xml
$(BIN)/gocov-xml:
	$(info $(M) building gocov-xml…)
	$Q go get github.com/AlekSi/gocov-xml

GO2XUNIT = $(BIN)/go2xunit
$(BIN)/go2xunit:
	$(info $(M) building go2xunit…)
	$Q go get github.com/tebeka/go2xunit

# Tests

TEST_TARGETS := test-default test-bench test-short test-verbose test-race
.PHONY: $(TEST_TARGETS) test-xml check test tests
test-bench:   ARGS=-run=__absolutelynothing__ -bench=. ## Run benchmarks
test-short:   ARGS=-short        ## Run only short tests
test-verbose: ARGS=-v            ## Run tests in verbose mode with coverage reporting
test-race:    ARGS=-race         ## Run tests with race detector
$(TEST_TARGETS): NAME=$(MAKECMDGOALS:test-%=%)
$(TEST_TARGETS): test
check test tests: fmt lint sigupdate sigserver; $(info $(M) running $(NAME:%=% )tests…) @ ## Run tests
	$Q cd $(BASE) && $(GO) test -timeout $(TIMEOUT)s $(ARGS) $(TESTPKGS)

test-xml: fmt lint sigupdate sigserver $(GO2XUNIT) ; $(info $(M) running $(NAME:%=% )tests…) @ ## Run tests with xUnit output
	$Q cd $(BASE) && 2>&1 $(GO) test -timeout 20s -v $(TESTPKGS) | tee test/tests.output
	$(GO2XUNIT) -fail -input test/tests.output -output test/tests.xml

COVERAGE_MODE = atomic
COVERAGE_PROFILE = $(COVERAGE_DIR)/profile.out
COVERAGE_XML = $(COVERAGE_DIR)/coverage.xml
COVERAGE_HTML = $(COVERAGE_DIR)/index.html
.PHONY: test-coverage test-coverage-tools
test-coverage-tools: | $(GOCOVMERGE) $(GOCOV) $(GOCOVXML)
test-coverage: COVERAGE_DIR := $(CURDIR)/test/coverage.$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
test-coverage: fmt lint vendor test-coverage-tools | $(BASE) ; $(info $(M) running coverage tests…) @ ## Run coverage tests
	$Q mkdir -p $(COVERAGE_DIR)/coverage
	$Q cd $(BASE) && for pkg in $(TESTPKGS); do \
		$(GO) test \
			-coverpkg=$$($(GO) list -f '{{ join .Deps "\n" }}' $$pkg | \
					grep '^$(PACKAGE)/' | grep -v '^$(PACKAGE)/vendor/' | \
					tr '\n' ',')$$pkg \
			-covermode=$(COVERAGE_MODE) \
			-coverprofile="$(COVERAGE_DIR)/coverage/`echo $$pkg | tr "/" "-"`.cover" $$pkg ;\
	 done
	$Q $(GOCOVMERGE) $(COVERAGE_DIR)/coverage/*.cover > $(COVERAGE_PROFILE)
	$Q $(GO) tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	$Q $(GOCOV) convert $(COVERAGE_PROFILE) | $(GOCOVXML) > $(COVERAGE_XML)

.PHONY: lint
lint: vendor $(GOLINT) ; $(info $(M) running golint…) @ ## Run golint
	$Q cd $(BASE) && ret=0 && for pkg in $(PKGS); do \
		test -z "$$($(GOLINT) $$pkg | tee /dev/stderr)" || ret=1 ; \
	 done ; exit $$ret

.PHONY: fmt
fmt: vendor; $(info $(M) running gofmt…) @ ## Run gofmt on all source files
	@ret=0 && for d in $$($(GO) list -f '{{.Dir}}' ./... | grep -v /vendor/); do \
		$(GOFMT) -l -w $$d/*.go || ret=$$? ; \
	 done ; exit $$ret

sigserver: vendor
	$(info $(M) building sigserver…) @ ## Build sigupdate binary
	$Q cd $(BASE) && $(GO) build \
		-tags release \
		-ldflags '-X main.githash=$(GITHASH) -X main.buildstamp=$(DATE) -X main.appversion=$(VERSION)' \
		-o $(CURDIR)/bin/sigserver \
        $(GOPATH)/src/$(PACKAGE)/sigserver/app/*.go

sigupdate: vendor
	$(info $(M) building sigupdate…) @ ## Build sigserver binary
	$Q cd $(BASE) && $(GO) build \
		-tags release \
		-ldflags '-X main.githash=$(GITHASH) -X main.buildstamp=$(DATE) -X main.appversion=$(VERSION)' \
		-o $(CURDIR)/bin/sigupdate \
        $(GOPATH)/src/$(PACKAGE)/sigupdate/app/*.go

# Dependency management

glide.lock: glide.yaml; $(info $(M) updating dependencies…)
	$Q cd $(BASE) && $(GLIDE) update
	@touch $@

vendor: $(BASE) glide.lock; $(info $(M) retrieving dependencies…)
	$Q cd $(BASE) && $(GLIDE) --quiet install
	@ln -nsf . vendor/src
	@touch $@
	@cp -Ra $(CURDIR)/vendor/* $(GOPATH)/src/

# Misc

.PHONY: clean
clean: ; $(info $(M) cleaning…)	@ ## Cleanup everything
	@rm -rf $(GOPATH)
	@rm -rf bin
	@rm -rf test/tests.* test/coverage.*

.PHONY: help
help:
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: version
version:
	@echo $(VERSION)

release: all
	@gzip -9 < $(CURDIR)/bin/sigserver > $(CURDIR)/bin/sigserver-$(VERSION)-$(PLATFORM)-$(ARCH).gz
	@gzip -9 <$(CURDIR)/bin/sigupdate > $(CURDIR)/bin/sigupdate-$(VERSION)-$(PLATFORM)-$(ARCH).gz
	@echo "sigserver-$(VERSION)-$(PLATFORM)-$(ARCH).gz SHA256 `sha256sum $(CURDIR)/bin/sigserver-$(VERSION)-$(PLATFORM)-$(ARCH).gz | cut -f1 -d' '`"
	@echo "sigserver-$(VERSION)-$(PLATFORM)-$(ARCH).gz MD5 `md5sum $(CURDIR)/bin/sigserver-$(VERSION)-$(PLATFORM)-$(ARCH).gz | cut -f1 -d' '`"
	@echo "sigupdate-$(VERSION)-$(PLATFORM)-$(ARCH).gz SHA256 `sha256sum $(CURDIR)/bin/sigupdate-$(VERSION)-$(PLATFORM)-$(ARCH).gz | cut -f1 -d' '`"
	@echo "sigupdate-$(VERSION)-$(PLATFORM)-$(ARCH).gz MD5 `md5sum $(CURDIR)/bin/sigupdate-$(VERSION)-$(PLATFORM)-$(ARCH).gz | cut -f1 -d' '`"
	@sed -i "s/^ENV SIGSERVER_VERSION .*/ENV SIGSERVER_VERSION $(VERSION)/" $(CURDIR)/Dockerfile
	@sed -i "s/^ENV SIGSERVER_SHA256SUM .*/ENV SIGSERVER_SHA256SUM `sha256sum $(CURDIR)/bin/sigserver-$(VERSION)-$(PLATFORM)-$(ARCH).gz | cut -f1 -d' '`/" $(CURDIR)/Dockerfile
