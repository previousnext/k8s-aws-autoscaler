package main

import (
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/previousnext/k8s-aws-autoscaler/cmd"
)

func main() {
	app := kingpin.New("k8s-aws-autoscaler", "Kubernetes AWS Scaler: Deployments")

	cmd.Watch(app)
	cmd.Version(app)

	kingpin.MustParse(app.Parse(os.Args[1:]))
}
