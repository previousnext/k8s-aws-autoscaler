#!/usr/bin/make -f

export CGO_ENABLED=0

PROJECT=github.com/previousnext/k8s-aws-autoscaler
VERSION=$(shell git describe --tags --always)
COMMIT=$(shell git rev-list -1 HEAD)

# Builds the project.
build:
	gox -os='linux darwin' \
	    -arch='amd64' \
	    -output='bin/k8s-aws-autoscaler_{{.OS}}_{{.Arch}}' \
	    -ldflags='-extldflags "-static" -X github.com/previousnext/k8s-aws-autoscaler/cmd.GitVersion=${VERSION} -X github.com/previousnext/k8s-aws-autoscaler/cmd.GitCommit=${COMMIT}' \
        $(PROJECT)

# Run all lint checking with exit codes for CI
lint:
	golint -set_exit_status `go list ./... | grep -v /vendor/`

# Run tests with coverage reporting
test:
	go test -cover ./...

IMAGE=previousnext/k8s-aws-autoscaler

# Releases the project Docker Hub
release:
	docker build -t ${IMAGE}:${VERSION} .
	docker push ${IMAGE}:${VERSION}

.PHONY: build lint test release
