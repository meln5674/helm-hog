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
	testBatch              bool
	testOnlyLint           bool
	testNoApply            bool
	testParallel           int
	testKeepReports        bool
	testPruneFailedChoices bool
	testAutoRemoveSuccess  bool
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

		fmt.Printf("Reports will be kept at %s\n", loadedProject.TempDir)

		failed := make([]helmhog.Case, 0)
		skipped := make([]helmhog.Case, 0)

		failedVariables := make(map[helmhog.VariableName]map[helmhog.ChoiceName]struct{}, len(loadedProject.Variables))
		for k, v := range loadedProject.Variables {
			failedVariables[k] = make(map[helmhog.ChoiceName]struct{}, len(v))
		}

		type result struct {
			err     error
			c       helmhog.Case
			skipped bool
		}

		results := make(chan result)

		if testParallel == 0 {
			testParallel = runtime.NumCPU()
		}
		workerSem := make(chan struct{}, testParallel)

		worker := func() {
			defer func() { workerSem <- struct{}{} }()
			for c := range cases {
				skipped, err := func() (bool, error) {
					if testPruneFailedChoices {
						for k, v := range c {
							if _, ok := failedVariables[k][v]; ok {
								return true, nil
							}
						}
					}
					err := loadedProject.MakeCaseTempDir(c)
					if err != nil {
						return false, errors.Wrap(err, fmt.Sprintf("create temp dir for case %v", c))
					}
					if testOnlyLint {
						err = loadedProject.Lint(c).Run()
					} else if testNoApply {
						err = loadedProject.Validate(c).Run()
					} else {
						err = loadedProject.ValidateWithApply(c).Run()
					}
					if testAutoRemoveSuccess {
						os.RemoveAll(loadedProject.TempPath(c))
						fmt.Printf("Removed %s\n", loadedProject.TempPath(c))
					} else {
						fmt.Printf("Not removing %s\n", loadedProject.TempPath(c))
					}
					if err == nil {
						return false, err
					}
					writeErr := os.WriteFile(loadedProject.TempPath(c, "err"), []byte(err.Error()), 0600)
					if writeErr != nil {
						return false, errors.Wrap(err, fmt.Sprintf("write error file for case %v: %v", c, err))
					}
					return false, err
				}()
				results <- result{c: c, err: err, skipped: skipped}
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

		resultCount := 0
		for result := range results {
			if result.err != nil {
				failed = append(failed, result.c)
				for k, v := range result.c {
					failedVariables[k][v] = struct{}{}
				}
				if len(failed) != 0 && len(failed)%10 == 0 {
					fmt.Printf("%d cases failed\n", len(failed))
				}
			}
			if result.skipped {
				skipped = append(skipped, result.c)
				if len(skipped)%10 == 0 {
					fmt.Printf("%d cases skipped\n", len(skipped))
				}
			} else {
				resultCount++
				if resultCount%10 == 0 {
					fmt.Printf("%d cases completed\n", resultCount)
				}
			}
		}

		if len(failed) == 0 && len(skipped) == 0 {
			fmt.Println("All cases passed!")
			return nil
		}

		fmt.Println("The following choice mappings had failed cases")
		for k, v := range failedVariables {
			fmt.Printf("%s:\n", k)
			for c := range v {
				fmt.Printf("- %s\n", c)
			}
		}

		fmt.Println("The following cases failed:")
		for _, c := range failed {
			fmt.Println(loadedProject.TempPath(c))
		}
		fmt.Println("The following cases were skipped:")
		for _, c := range skipped {
			fmt.Println(loadedProject.TempPath(c))
		}
		if testBatch {
			err = fmt.Errorf("Some tests failed or were skipped!")
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
	testCmd.Flags().BoolVar(&testOnlyLint, "only-lint", false, "If set, do not attempt to do a helm template or kubectl apply --dry-run")
	testCmd.Flags().BoolVar(&testNoApply, "no-apply", false, "If set, do not attempt to do a kubectl apply --dry-run, but still perform a helm template")
	testCmd.Flags().IntVar(&testParallel, "parallel", 1, "Number of cases to run in parallel. Set to zero to use number of cpu cores")
	testCmd.Flags().BoolVar(&testKeepReports, "keep-reports", false, "Do not delete reports, even if all cases pass")
	testCmd.Flags().BoolVar(&testPruneFailedChoices, "prune-failed-choices", false, "If true, skip any cases that share any choices with any failed cases. Note this is not guarnateed for performance reasons, and a few cases may still execute.")
	testCmd.Flags().BoolVar(&testAutoRemoveSuccess, "auto-remove-success", false, "If true, remove output files from successful cases immediately after case completion")
}
