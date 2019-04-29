DOCKER_ID_USER = simonpasquier

.PHONY: build
build:
	promu build -v

.PHONY: docker
docker: build
	@echo "Updating the local Docker image"
	docker build -t instrumented_app:latest .

.PHONY: pushimage
pushimage: docker
	@echo "Pushing image to $(DOCKER_ID_USER)/instrumented_app"
	docker tag instrumented_app:latest $(DOCKER_ID_USER)/instrumented_app
	docker push $(DOCKER_ID_USER)/instrumented_app
