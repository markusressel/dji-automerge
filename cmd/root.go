package cmd

import (
	"dji-automerge/internal"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"regexp"
)

var Input string
var Output string
var Filters []string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "dji-automerge",
	Short: "A small utility to automatically detect and join video segments from DJI drones.",
	Long:  `A small utility to automatically detect and join video segments from DJI drones.`,
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

		// check if filters are valid regex patterns
		for _, filter := range Filters {
			_, err := regexp.Compile(filter)
			if err != nil {
				return errors.Join(errors.New("invalid filter \""+filter+"\", must be a valid regex pattern"), err)
			}
		}

		params := internal.ProcessingParams{
			InputPath:  input,
			OutputPath: output,
			Filters:    Filters,
		}
		return internal.Process(params)
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

	RootCmd.PersistentFlags().StringArrayVarP(
		&Filters,
		"filter", "f",
		[]string{},
		"Filters to apply to the video files")

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
