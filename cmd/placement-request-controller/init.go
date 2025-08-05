package main

import "flag"

func init() {
	flag.StringVar(&KubeConfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
}
