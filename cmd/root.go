package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pingo",
	Short: "pingo your ping in Go",
	Long:  "pingo is a Go implementation of the ping utility",
}

func Execute() error {
	return rootCmd.Execute()
}
