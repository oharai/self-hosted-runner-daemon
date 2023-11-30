package main

import "github.com/oharai/self-hosted-runner-daemon/cmd"

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		panic(err)
	}
}
