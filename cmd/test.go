/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/meln5674/helm-hog/pkg/helmhog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	testBatch    bool
	testOnlyLint bool
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cases := make(chan helmhog.Case)

		if !testBatch {
			defer os.RemoveAll(loadedProject.TempDir)
		}

		defer func() {
			for range cases {
			}
		}()

		go loadedProject.GenerateCases(cases)

		failed := make([]helmhog.Case, 0)

		type foo struct{}

		for c := range cases {
			err := loadedProject.MakeCaseTempDir(c)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("create temp dir for case %v", c))
			}
			if testOnlyLint {
				err = loadedProject.Lint(c).Run()
			} else {
				err = loadedProject.Validate(c).Run()
			}
			if err != nil {
				failed = append(failed, c)
				err = os.WriteFile(loadedProject.TempPath(c, "err"), []byte(err.Error()), 0600)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("write error file for case %v", c))
				}
			}
		}

		if len(failed) == 0 {
			fmt.Println("All cases passed!")
			return nil
		}

		fmt.Println("The following cases failed:")
		for _, c := range failed {
			fmt.Println(c)
		}
		if testBatch {
			fmt.Printf("Reports are found at %s, user is responsible for deleting this directory\n", loadedProject.TempDir)
			return fmt.Errorf("Some tests failed!")
		}
		fmt.Printf("Reports are found at %s, press enter when ready to remove (pass --batch to not delete report directories)\n", loadedProject.TempDir)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.Flags().BoolVar(&testBatch, "batch", false, "If set, do not prompt the user for report cleanup, and return non-zero on failure.")
	testCmd.Flags().BoolVar(&testOnlyLint, "only-lint", false, "If set, do not attempt to do a kubectl apply --dry-run")
}
