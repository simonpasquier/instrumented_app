DOCKER_ID_USER = simonpasquier

build:
	go build .

buildstatic:
	go build -tags netgo .

docker: buildstatic
	@echo "Updating the local Docker image"
	docker build -t instrumented_app:latest .

pushimage: docker
	@echo "Pushing image to $(DOCKER_ID_USER)/instrumented_app"
	docker tag instrumented_app:latest $(DOCKER_ID_USER)/instrumented_app
	docker push $(DOCKER_ID_USER)/instrumented_app
