package main

import "github.com/ncxton/potaco/internal/cli"

var version = "unknown"

func main() {
	cli.SetVersion(version)
	cli.Execute()
}
