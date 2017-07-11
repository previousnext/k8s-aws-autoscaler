#!/usr/bin/make -f

build:
	./hack/build.sh linux server k8s-aws-autoscaler github.com/previousnext/k8s-aws-autoscaler

lint:
	golint -set_exit_status src/github.com/previousnext/k8s-aws-autoscaler/...

test:
	./hack/test.sh github.com/previousnext/k8s-aws-autoscaler/...
