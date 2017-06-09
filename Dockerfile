FROM golang:1.8
ADD . /go
RUN make build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=0 /go/bin/linux/server/k8s-aws-autoscaler /usr/local/bin/k8s-aws-autoscaler
CMD ["k8s-aws-autoscaler"]
