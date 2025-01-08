package main

import (
	"flag"
	"net"
)

type CommandLineArguments struct {
	BindAddress    string
	InsertionLabel string
}

func (args *CommandLineArguments) BindAddressIsInvalid() bool {
	if _, err := net.ResolveTCPAddr("tcp4", args.BindAddress); err != nil {
		return true
	}

	return false
}

func ParseCommandLineArguments() *CommandLineArguments {
	cliArgs := CommandLineArguments{}

	flag.StringVar(&cliArgs.BindAddress, "bindAddr", "127.0.0.1:9091", "local bind address for proxy")
	flag.StringVar(&cliArgs.InsertionLabel, "insertionLabel", "", "insertion label to add to the proxy messages")
	flag.Parse()

	return &cliArgs
}
