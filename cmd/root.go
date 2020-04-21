package cmd

import (
	"fmt"

	"github.com/mikaelmello/pingo/core"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pingo",
	Short: "pingo your ping in Go",
	Long:  "pingo is a Go implementation of the ping utility",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		bundle, err := core.NewBundle(args[0])
		if err != nil {
			fmt.Printf("Error %s\n", err.Error())
			return
		}

		err = bundle.Start()
		if err != nil {
			fmt.Printf("Error %s\n", err.Error())
			return
		}

	},
}

// Execute executes the root command of the application.
func Execute() error {
	return rootCmd.Execute()
}
