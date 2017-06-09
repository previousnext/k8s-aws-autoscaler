#!/usr/bin/make -f

build:
	./hack/build.sh linux server k8s-aws-autoscaler github.com/previousnext/k8s-aws-autoscaler

image:
	docker build -t previousnext/k8s-aws-autoscaler:latest .

.PHONY: build image
