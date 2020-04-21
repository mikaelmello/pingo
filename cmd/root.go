package cmd

import (
	"errors"
	"fmt"

	"github.com/mikaelmello/pingo/core"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pingo",
	Short: "pingo your ping in Go",
	Long:  "pingo is a Go implementation of the ping utility",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("You must provide a hostname or IP address")
		}

		if !core.IsValidIPAddressOrHostname(args[0]) {
			return fmt.Errorf("Invalid hostname or IP address specified: %s", args[0])
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello, World!")
	},
}

// Execute executes the root command of the application.
func Execute() error {
	return rootCmd.Execute()
}
