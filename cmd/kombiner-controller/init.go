package main

import "flag"

func init() {
	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. ")
}
