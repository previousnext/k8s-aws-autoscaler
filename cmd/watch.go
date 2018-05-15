package cmd

import (
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/previousnext/k8s-aws-autoscaler/internal/scaler"
)

type cmdWatch struct {
	params scaler.WatchParams
}

func (cmd *cmdWatch) run(c *kingpin.ParseContext) error {
	return scaler.Watch(os.Stdout, cmd.params)
}

// Watch declares the "watch" sub command.
func Watch(app *kingpin.Application) {
	c := new(cmdWatch)

	cmd := app.Command("watch", "Watch to capacity changes").Action(c.run)
	cmd.Flag("group", "The Autoscaling group to update periodically").Required().Envar("GROUP").StringVar(&c.params.Group)
	cmd.Flag("frequency", "How often to run the check").Default("120s").Envar("FREQUENCY").DurationVar(&c.params.Frequency)
	cmd.Flag("scale-down-timeout", "How long to wait before scaling down (in minutes)").Default("60").Envar("SCALE_DOWN_TIMEOUT").Float64Var(&c.params.DownTimeout)
	cmd.Flag("dry", "Don't make any changes!").BoolVar(&c.params.DryRun)
	cmd.Flag("node-cpu", "Declare how much cpu the node has in the scaling group").Default("200").Envar("NODE_CPU").IntVar(&c.params.NodeCPU)
	cmd.Flag("node-mem", "Declare how much memory the node has in the scaling group").Default("7000").Envar("NODE_MEM").IntVar(&c.params.NodeMemory)
}
