package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var (
	debug bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kube-timer",
	Short: "Get the provision/deletion times of the kubernetes resources.",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatalf("Error executing root command: %v", err)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "", false, "Enable debug logging")
}
