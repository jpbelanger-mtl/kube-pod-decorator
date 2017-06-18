PACKAGE_NAME=github.com/jpbelanger-mtl/kube-pod-decorator
BUILD_IMAGE_TAG=kube-pod-decorator/build:build
BUILD_CONTAINER_NAME=kube-pod-decorator-build
BIN_NAME=kube-pod-decorator
BIN_PACKAGE=.

clean:
	rm -rf dist
dist:
	mkdir dist
build: dist
	docker build --no-cache -t $(BUILD_IMAGE_TAG) --build-arg PROJECT_NAME=$(PACKAGE_NAME) --build-arg BIN_PKG=$(BIN_PACKAGE) -f build.Dockerfile .
	docker rm --force $(BUILD_CONTAINER_NAME) || true
	docker create --name $(BUILD_CONTAINER_NAME) $(BUILD_IMAGE_TAG)
	docker cp $(BUILD_CONTAINER_NAME):/go/bin/$(BIN_NAME) dist
	docker rm -f $(BUILD_CONTAINER_NAME)
	docker rmi $(BUILD_IMAGE_TAG)
build-docker: clean build
	docker build --no-cache .
build-dev:
	go install $(PACKAGE_NAME)/$(BIN_PACKAGE)