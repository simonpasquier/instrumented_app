DOCKER_ID_USER = simonpasquier
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H-%M-%SZ)
COMMIT_ID := $(shell git log --pretty=format:'%h' -n 1)
GO_FLAGS := -ldflags "-X main.buildDate=$(BUILD_DATE) -X main.commitId=$(COMMIT_ID)"

build:
	go build $(GO_FLAGS) .

buildstatic:
	go build $(GO_FLAGS) -tags netgo .

docker: buildstatic
	@echo "Updating the local Docker image"
	docker build -t instrumented_app:latest .

pushimage: docker
	@echo "Pushing image to $(DOCKER_ID_USER)/instrumented_app"
	docker tag instrumented_app:latest $(DOCKER_ID_USER)/instrumented_app
	docker push $(DOCKER_ID_USER)/instrumented_app
