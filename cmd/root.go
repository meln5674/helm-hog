/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"sigs.k8s.io/yaml"

	"github.com/meln5674/helm-hog/pkg/helmhog"
)

var (
	projectPath string

	project       *helmhog.Project
	loadedProject *helmhog.LoadedProject
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:               "helm-hog",
	Short:             "Clean your Helm Charts",
	Long:              `Helm Hog lets you generate and automatically validate many combinations of values for your helm charts`,
	PersistentPreRunE: loadProject,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&projectPath, "project", "hog.yaml", "Path to project YAML file")
}

func loadProject(*cobra.Command, []string) error {
	project = new(helmhog.Project)
	projectBytes, err := os.ReadFile(projectPath)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("read file %s", projectPath))
	}
	err = yaml.Unmarshal(projectBytes, project)
	if err != nil {
		return errors.Wrap(err, "parse project yaml")
	}
	loadedProject, err = project.Load()
	if err != nil {
		return errors.Wrap(err, "invalid project")
	}
	return nil
}
