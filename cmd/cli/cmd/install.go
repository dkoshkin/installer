package cmd

import (
	"github.com/mesosphere/installer/pkg/executor"
	"github.com/mesosphere/installer/pkg/util"
	"github.com/spf13/cobra"
)

type installOptions struct {
	inventoryFile      string
	configurationFile  string
	generatedAssetsDir string
	verboseLevel       int
	nodes              []string
}

var installOpts installOptions

// installCmd represents the init command
var installCmd = &cobra.Command{
	SilenceUsage: true,
	SilenceErrors: true,
	Use:          "install",
	RunE: func(cmd *cobra.Command, args []string) error {

		executorOpts := executor.ExecutorOptions{
			GeneratedAssetsDirectory: installOpts.generatedAssetsDir,
			VerboseLevel:             installOpts.verboseLevel,
		}

		ansibleExecutor, err := executor.NewExecutor(out, errOut, executorOpts)
		if err != nil {
			return err
		}
		err = ansibleExecutor.Install(installOpts.inventoryFile, installOpts.configurationFile, installOpts.nodes...)
		if err != nil {
			return err
		}
		util.PrintColor(out, util.Green,"Kubernetes Cluster Installed Successfully!\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	installCmd.Flags().StringVar(&installOpts.inventoryFile, "inventory", "inventory.ini", "path to the inventory.ini file")
	installCmd.Flags().StringVar(&installOpts.configurationFile, "configuration", "configuration.yaml", "path to the configuration.yaml to override defaults")
	installCmd.Flags().StringVar(&installOpts.generatedAssetsDir, "generated-assets-dir", "generated", "path to the directory where assets generated during the installation process will be stored")
	installCmd.Flags().IntVar(&installOpts.verboseLevel, "verbose", 1, "logging level")
	installCmd.Flags().StringSliceVar(&installOpts.nodes, "limit", []string{}, "comma-separated list of hostnames to limit the execution to a subset of nodes")
}
