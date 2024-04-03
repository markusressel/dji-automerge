package cmd

import (
	"dji-automerge/internal"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var Input string
var Output string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "dji-automerge",
	Short: "Automatically join video parts.",
	Long:  `Automatically join video parts.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		input := Input
		output := Output

		if len(input) <= 0 {
			input, err = os.Getwd()
			if err != nil {
				return err
			}
		}
		input, err = filepath.Abs(input)
		if err != nil {
			return err
		}

		if len(output) <= 0 {
			output = input
		}
		output, err = filepath.Abs(output)
		if err != nil {
			return err
		}

		return internal.Process(input, output)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	RootCmd.PersistentFlags().StringVarP(
		&Input,
		"input", "i",
		"",
		"Input directory",
	)

	RootCmd.PersistentFlags().StringVarP(
		&Output,
		"output", "o",
		"",
		"Output directory",
	)

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {

}
