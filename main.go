package main

import (
	"context"
	"os"

	"github.com/bashfulrobot/configr/cmd/configr"
	"github.com/charmbracelet/fang"
)

func main() {
	cmd := configr.NewRootCmd()
	if err := fang.Execute(context.Background(), cmd); err != nil {
		os.Exit(1)
	}
}