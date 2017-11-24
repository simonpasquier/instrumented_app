build:
	go build .

buildstatic:
	go build -tags netgo .

docker: buildstatic
	docker build -t instrumented_app:latest .
