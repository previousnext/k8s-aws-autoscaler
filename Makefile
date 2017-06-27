#!/usr/bin/make -f

VERSION=$(shell git describe --tags --always)

release: build push

build:
	docker build -t previousnext/k8s-aws-autoscaler:${VERSION} .

push:
	docker push previousnext/k8s-aws-autoscaler:${VERSION}

.PHONY: build push
