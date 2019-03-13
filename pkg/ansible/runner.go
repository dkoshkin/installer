package ansible

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// OutputFormat is used for controlling the STDOUT format of the Ansible runner
type OutputFormat string

// Runner for running Ansible playbooks
type Runner interface {
	// StartPlaybook runs the playbook asynchronously with the given inventory and extra vars.
	StartPlaybook(playbookFileName string, inventoryFile string, configurationFile string) error
	// WaitPlaybook blocks until the execution of the playbook is complete. If an error occurred,
	// it is returned. Otherwise, returns nil to signal the completion of the playbook.
	WaitPlaybook() error
	// StartPlaybookOnNode runs the playbook asynchronously with the given inventory and extra vars
	// against the specific node.
	StartPlaybookOnNode(playbookFileName string, inventoryFile string, configurationFile string, node ...string) error
}

type runner struct {
	// Out is the stdout writer for the Ansible process
	out io.Writer
	// ErrOut is the stderr writer for the Ansible process
	errOut io.Writer
	// Output also gets logged to a file
	log          io.Writer
	verboseLevel int
	pythonPath   string
	ansibleDir   string
	runDir       string
	waitPlaybook func() error
}

// NewRunner returns a new runner for running Ansible playbooks.
func NewRunner(out, errOut, log io.Writer, verboseLevel int, ansibleDir string, runDir string) (Runner, error) {
	// Ansible depends on python 2.7 being installed and on the path as "python".
	// Validate that it is available
	if _, err := exec.LookPath("python"); err != nil {
		return nil, fmt.Errorf("Could not find 'python' in the PATH. Ensure that python 2.7 is installed and in the path as 'python'.")
	}

	ppath, err := getPythonPath()
	if err != nil {
		return nil, err
	}

	return &runner{
		out:          out,
		errOut:       errOut,
		log:          log,
		verboseLevel: verboseLevel,
		pythonPath:   ppath,
		ansibleDir:   ansibleDir,
		runDir:       runDir,
	}, nil
}

// WaitPlaybook blocks until the ansible process running the playbook exits.
// If the process exits with a non-zero status, it will return an error.
func (r *runner) WaitPlaybook() error {
	if r.waitPlaybook == nil {
		return fmt.Errorf("wait called, but playbook not started")
	}
	execErr := r.waitPlaybook()

	if execErr != nil {
		return fmt.Errorf("error running ansible: %v", execErr)
	}
	return nil
}

// RunPlaybook with the given inventory and extra vars
func (r *runner) StartPlaybook(playbookFileName string, inventoryFile string, configurationFile string) error {
	return r.startPlaybook(playbookFileName, inventoryFile, configurationFile) // Dson't set the --limit arg
}

// StartPlaybookOnNode runs the playbook asynchronously with the given inventory and extra vars against the specific node.
func (r *runner) StartPlaybookOnNode(playbookFileName string, inventoryFile string, configurationFile string, nodes ...string) error {
	return r.startPlaybook(playbookFileName, inventoryFile, configurationFile, nodes...) // Set the --limit arg to the node we want to target
}

func (r *runner) startPlaybook(playbookFileName string, inventoryFile string, configurationFile string, nodes ...string) error {
	playbook := filepath.Join(r.ansibleDir, "playbooks", playbookFileName)
	if _, err := os.Stat(playbook); os.IsNotExist(err) {
		return fmt.Errorf("playbook %q does not exist", playbook)
	}
	if err := copyFileContents(configurationFile, filepath.Join(r.runDir, "configuration.yaml")); err != nil {
		return fmt.Errorf("error copying configration.yaml to %q: %v", r.runDir, err)
	}
	if err := copyFileContents(inventoryFile, filepath.Join(r.runDir, "inventory.ini")); err != nil {
		return fmt.Errorf("error copying inventory.ini to %q: %v", r.runDir, err)
	}

	// TODO use provided directory
	pwd, _ := os.Getwd()

	cmd := exec.Command(filepath.Join(r.ansibleDir, "bin", "ansible-playbook"), "-i", configurationFile, "-s", playbook, "--extra-vars", fmt.Sprintf("install_directory=%s", pwd), "--extra-vars", "@"+configurationFile)
	// also log to a file
	outWriter := io.MultiWriter(r.log, r.out)
	errWriter := io.MultiWriter(r.log, r.errOut)
	cmd.Stdout = outWriter
	cmd.Stderr = errWriter
	cmd.Stdin = os.Stdin

	log.SetOutput(r.out)

	limitArg := strings.Join(nodes, ",")
	if limitArg != "" {
		cmd.Args = append(cmd.Args, "--limit", limitArg)
	}

	if r.verboseLevel > 0 {
		verboseLevel := fmt.Sprintf("-%s", strings.Repeat("v", r.verboseLevel))
		cmd.Args = append(cmd.Args, verboseLevel)
	}


	os.Setenv("PYTHONPATH", r.pythonPath)
	os.Setenv("ANSIBLE_CONFIG", filepath.Join(r.ansibleDir, "playbooks", "ansible.cfg"))

	// Print Ansible command
	fmt.Fprintf(r.log, "export PYTHONPATH=%v\n", os.Getenv("PYTHONPATH"))
	fmt.Fprintf(r.log, "export ANSIBLE_CONFIG=%v\n", os.Getenv("ANSIBLE_CONFIG"))
	fmt.Fprintln(r.log, strings.Join(cmd.Args, " "))

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("error running playbook: %v", err)
	}

	r.waitPlaybook = cmd.Wait

	return nil
}

func getPythonPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting working dir: %v", err)
	}
	lib := filepath.Join(wd, "ansible", "lib", "python2.7", "site-packages")
	lib64 := filepath.Join(wd, "ansible", "lib64", "python2.7", "site-packages")
	return fmt.Sprintf("%s:%s", lib, lib64), nil
}

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
