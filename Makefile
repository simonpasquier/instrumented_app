FIRST_GOPATH      := $(firstword $(subst :, ,$(shell go env GOPATH)))
GO_BUILD_PLATFORM ?= linux-amd64
PREFIX            ?= $(shell pwd)
PROMU             := $(FIRST_GOPATH)/bin/promu
PROMU_VERSION     ?= 0.5.0
PROMU_URL         := https://github.com/prometheus/promu/releases/download/v$(PROMU_VERSION)/promu-$(PROMU_VERSION).$(GO_BUILD_PLATFORM).tar.gz

.PHONY: build
build: promu
	$(PROMU) build -v --prefix $(PREFIX)

.PHONY: promu
promu: $(PROMU)

$(PROMU):
	$(eval PROMU_TMP := $(shell mktemp -d))
	curl -s -L $(PROMU_URL) | tar -xvzf - -C $(PROMU_TMP)
	mkdir -p $(FIRST_GOPATH)/bin
	cp $(PROMU_TMP)/promu-$(PROMU_VERSION).$(GO_BUILD_PLATFORM)/promu $(FIRST_GOPATH)/bin/promu
	rm -r $(PROMU_TMP)

.PHONY: container-build
container-build: build
	@echo "Updating the local container image"
	docker build -t instrumented_app:latest .

.PHONY: push
push: container-build
	@echo "Pushing image to quay.io/simonpasquier/instrumented_app"
	docker tag instrumented_app:latest quay.io/simonpasquier/instrumented_app
	docker push quay.io/simonpasquier/instrumented_app
