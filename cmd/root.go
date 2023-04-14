/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	goflag "flag"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"sigs.k8s.io/yaml"

	"github.com/meln5674/helm-hog/pkg/helmhog"
)

var (
	projectPath string

	project       *helmhog.Project
	loadedProject *helmhog.LoadedProject

	helmFlags    []string
	kubectlFlags []string
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
	rootCmd.PersistentFlags().StringSliceVar(&helmFlags, "helm-flags", []string{}, "Extra flags to pass to the helm command")
	rootCmd.PersistentFlags().StringSliceVar(&kubectlFlags, "kubectl-flags", []string{}, "Extra flags to pass to the kubectl command")

	klogFlags := goflag.NewFlagSet("", goflag.PanicOnError)
	klog.InitFlags(klogFlags)
	rootCmd.PersistentFlags().AddGoFlagSet(klogFlags)
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
	loadedProject, err = project.Load(helmhog.ProjectSettings{
		KubectlFlags: kubectlFlags,
		HelmFlags:    helmFlags,
	})
	if err != nil {
		return errors.Wrap(err, "invalid project")
	}
	return nil
}
