/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"runtime"

	"github.com/meln5674/helm-hog/pkg/helmhog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	testBatch       bool
	testOnlyLint    bool
	testParallel    int
	testKeepReports bool
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
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		cases := make(chan helmhog.Case)

		defer func() {
			if testKeepReports || (testBatch && err == nil) {
				fmt.Printf("Reports are found at %s , user is responsible for deleting this directory\n", loadedProject.TempDir)
				return
			}
			os.RemoveAll(loadedProject.TempDir)
		}()

		defer func() {
			for range cases {
			}
		}()

		go loadedProject.GenerateCases(cases)

		failed := make([]helmhog.Case, 0)

		type result struct {
			err error
			c   helmhog.Case
		}

		results := make(chan result)

		if testParallel == 0 {
			testParallel = runtime.NumCPU()
		}
		workerSem := make(chan struct{}, testParallel)

		worker := func() {
			defer func() { workerSem <- struct{}{} }()
			for c := range cases {
				err := func() error {
					err := loadedProject.MakeCaseTempDir(c)
					if err != nil {
						return errors.Wrap(err, fmt.Sprintf("create temp dir for case %v", c))
					}
					if testOnlyLint {
						err = loadedProject.Lint(c).Run()
					} else {
						err = loadedProject.Validate(c).Run()
					}
					if err == nil {
						return err
					}
					writeErr := os.WriteFile(loadedProject.TempPath(c, "err"), []byte(err.Error()), 0600)
					if writeErr != nil {
						return errors.Wrap(err, fmt.Sprintf("write error file for case %v: %v", c, err))
					}
					return err
				}()
				results <- result{c: c, err: err}
			}
		}

		for i := 0; i < testParallel; i++ {
			go worker()
		}
		go func() {
			for i := 0; i < testParallel; i++ {
				<-workerSem
			}
			close(results)
		}()

		for result := range results {
			if result.err != nil {
				failed = append(failed, result.c)
			}
		}

		if len(failed) == 0 {
			fmt.Println("All cases passed!")
			return nil
		}

		fmt.Println("The following cases failed:")
		for _, c := range failed {
			fmt.Println(loadedProject.TempPath(c))
		}
		if testBatch {
			err = fmt.Errorf("Some tests failed!")
			return
		}
		fmt.Printf("Reports are found at %s, press enter when ready to remove (use --keep-reports to not delete report directories. Use --batch to skip this prompt)\n", loadedProject.TempDir)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.Flags().BoolVar(&testBatch, "batch", false, "If set, do not prompt the user for report cleanup, and return non-zero on failure")
	testCmd.Flags().BoolVar(&testOnlyLint, "only-lint", false, "If set, do not attempt to do a kubectl apply --dry-run")
	testCmd.Flags().IntVar(&testParallel, "parallel", 1, "Number of cases to run in parallel. Set to zero to use number of cpu cores")
	testCmd.Flags().BoolVar(&testKeepReports, "keep-reports", false, "Do not delete reports, even if all cases pass")
}
