GIT_REV=$(shell git rev-parse --short HEAD)
.PHONY: build docker deploy

build:
	go build -ldflags="-X 'main.Version=$(GIT_REV)'" ./cmd/prom-k8s-versions

