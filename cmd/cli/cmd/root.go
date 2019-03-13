package cmd

import (
	"encoding/json"
	"github.com/mesosphere/installer/pkg/util"
	"os"

	"github.com/spf13/cobra"
)

type Version struct {
	Version   string `json:"Version"`
	BuildDate string `json:"BuildDate"`
}

type rootOptions struct {
	clusterName string
}

var out = os.Stdout
var errOut = os.Stderr

var rootOpts rootOptions

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "installer",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version Version) {
	// set version string
	bytes, _ := json.MarshalIndent(version, "", "    ")
	rootCmd.SetVersionTemplate(string(bytes) + "\n")
	// also need to set Version to get cobra to print it
	rootCmd.Version = version.Version
	if err := rootCmd.Execute(); err != nil {
		util.PrintColor(errOut, util.Red, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	installCmd.Flags().StringVar(&rootOpts.clusterName, "cluster-name", "kubernetes", "the name of the cluster")
}
