FIRST_GOPATH      := $(firstword $(subst :, ,$(shell go env GOPATH)))
GO_BUILD_PLATFORM ?= linux-amd64
PREFIX            ?= $(shell pwd)
PROMU             := $(FIRST_GOPATH)/bin/promu

.PHONY: build
build: promu
	promu build -v --prefix $(PREFIX)

.PHONY: promu
promu: $(PROMU)

$(PROMU):
	cd ..
	GO111MODULE=on GOOS= GOARCH= go install github.com/prometheus/promu@master

.PHONY: container-build
container-build: build
	@echo "Updating the local container image"
	docker build -t instrumented_app:latest .

.PHONY: push
push: container-build
	@echo "Pushing image to quay.io/simonpasquier/instrumented_app"
	docker tag instrumented_app:latest quay.io/simonpasquier/instrumented_app
	docker push quay.io/simonpasquier/instrumented_app
