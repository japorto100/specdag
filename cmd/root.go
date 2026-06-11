package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const Version = "0.4.0"

var rootCmd = &cobra.Command{
	Use:   "specdag",
	Short: "specdag validates, assembles and renders dependency maps",
	Long:  `A fast, portable CLI tool to enforce structural validation, cycle detection and context assembly for Event-Spec-Driven Development.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(assembleCmd)
	rootCmd.AddCommand(renderCmd)
	rootCmd.AddCommand(hashCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(summaryCmd)
	rootCmd.AddCommand(impactCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(checkCatalogsCmd)
}
