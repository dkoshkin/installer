package executor

import (
	"fmt"
	"github.com/mesosphere/installer/pkg/ansible"
	"io"
	"os"
	"path/filepath"
	"time"
)

// The Executor will carry out the installation plan
type Executor interface {
	Install(inventoryFile string, configurationFile string, nodes ...string) error
}

// ExecutorOptions are used to configure the executor
type ExecutorOptions struct {
	// GeneratedAssetsDirectory is the location where generated assets re to be stored
	GeneratedAssetsDirectory string
	// Verbose output level from the executor
	VerboseLevel int
	// RunsDirectory is where information about installation runs is kept
	RunsDirectory string
	// DryRun determines if the executor should actually run the task
	DryRun bool
}

// NewExecutor returns an executor for performing installations according to the installation plan.
func NewExecutor(out io.Writer, errOut io.Writer, options ExecutorOptions) (Executor, error) {
	ansibleDir := "ansible"
	if options.GeneratedAssetsDirectory == "" {
		return nil, fmt.Errorf("GeneratedAssetsDirectory option cannot be empty")
	}
	if options.RunsDirectory == "" {
		options.RunsDirectory = "./runs"
	}

	certsDir := filepath.Join(options.GeneratedAssetsDirectory, "pki")

	return &ansibleExecutor{
		options:    options,
		out:        out,
		errOut:     errOut,
		ansibleDir: ansibleDir,
		certsDir:   certsDir,
	}, nil
}

type ansibleExecutor struct {
	options    ExecutorOptions
	out        io.Writer
	errOut     io.Writer
	ansibleDir string
	certsDir   string
}

type task struct {
	// name of the task used for the runs dir
	name string
	// the inventory of nodes to use
	inventoryFile string
	// the user provided configuration
	configurationFile string
	// the playbook filename
	playbook string
	// run the task on specific nodes
	limit []string
}

// execute will run the given task, and setup all what's needed for us to run ansible.
func (ae *ansibleExecutor) execute(t task) error {
	runDirectory, err := ae.createRunDirectory(t.name)
	if err != nil {
		return fmt.Errorf("error creating working directory for %q: %v", t.name, err)
	}
	ansibleLogFilename := filepath.Join(runDirectory, "ansible.log")
	ansibleLogFile, err := os.Create(ansibleLogFilename)
	defer ansibleLogFile.Close()
	if err != nil {
		return fmt.Errorf("error creating ansible log file %q: %v", ansibleLogFilename, err)
	}
	runner, err := ae.ansibleRunner(ansibleLogFile, runDirectory)
	if err != nil {
		return err
	}

	// Start running ansible with the given playbook
	if t.limit != nil && len(t.limit) != 0 {
		err = runner.StartPlaybookOnNode(t.playbook, t.inventoryFile, t.configurationFile, t.limit...)
	} else {
		err = runner.StartPlaybook(t.playbook, t.inventoryFile, t.configurationFile)
	}
	if err != nil {
		return fmt.Errorf("error running ansible playbook: %v", err)
	}

	// Wait until ansible exits
	if err = runner.WaitPlaybook(); err != nil {
		return fmt.Errorf("error running ansible playbook: %v", err)
	}
	return nil
}

// Install the cluster
func (ae *ansibleExecutor) Install(inventoryFile string, configurationFile string, nodes ...string) error {
	// Build the ansible inventory
	t := task{
		name:              "install",
		playbook:          "install.yaml",
		inventoryFile:     inventoryFile,
		configurationFile: configurationFile,
		limit:             nodes,
	}
	return ae.execute(t)
}

func (ae *ansibleExecutor) createRunDirectory(runName string) (string, error) {
	start := time.Now()
	runDirectory := filepath.Join(ae.options.RunsDirectory, runName, start.Format("2006-01-02-15-04-05"))
	if err := os.MkdirAll(runDirectory, 0777); err != nil {
		return "", fmt.Errorf("error creating directory: %v", err)
	}
	return runDirectory, nil
}

func (ae *ansibleExecutor) ansibleRunner(ansibleLog io.Writer, runDirectory string) (ansible.Runner, error) {
	// Send stdout and stderr to ansibleOut
	runner, err := ansible.NewRunner(ae.out, ae.errOut, ansibleLog, ae.options.VerboseLevel, ae.ansibleDir, runDirectory)
	if err != nil {
		return nil, fmt.Errorf("error creating ansible runner: %v", err)
	}

	return runner, nil
}
