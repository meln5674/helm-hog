/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/meln5674/helm-hog/pkg/helmhog"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		cases := make(chan helmhog.Case)
		go loadedProject.GenerateCases(cases)

		for c := range cases {
			fmt.Println(c)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
