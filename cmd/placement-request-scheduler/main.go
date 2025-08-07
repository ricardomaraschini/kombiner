package main

import (
	"fmt"
	"os"

	"k8s.io/kubernetes/cmd/kube-scheduler/app"

	"placement-request-controller/pkg/scheduler"
)

func main() {
	command := app.NewSchedulerCommand(
		app.WithPlugin(scheduler.PluginName, scheduler.NewBindPlugin),
	)
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
