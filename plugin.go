package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type (
	Plugin struct {
		Action          string
		Async           bool
		ConfigTemplate  string
		CreatePolicy    string
		DeletePolicy    string
		Deployment      string
		Description     string
		Dryrun          bool
		GcloudCmd       string
		OutputFile      string
		Preview         bool
		Project         string
		Token           string //
		Vars            map[string]interface{}
		Verbose         bool
	}
)

// Exec executes the plugin step
func (p Plugin) Exec() error {
	if p.Verbose {
		fmt.Printf("%+v\n", p.Vars)
	}

	p, err := checkAndFixParams(p)
	if err != nil {
		return err
	}

	// gcloud auth activate service account 
	err = activateServiceAccount(p)
	if err != nil {
		return fmt.Errorf("error in activateServiceAccount: %s", err)
	}
	
	// interpolate the template
	interpolateTemplate(p)

	if p.Verbose {
		dumpFile(os.Stdout, "DEPLOYMENT CONFIGURATION", p.OutputFile)
	}

	deployArgs := []string{
		"--project",
		p.Project,
		"deployment-manager",
		"deployments",
		p.Action,
		p.Deployment,
		"--config",
		p.OutputFile,
	}

	if p.Preview {
		deployArgs = append(deployArgs, "--preview")
	}

	if p.Async {
		deployArgs = append(deployArgs, "--async")
	}

	if p.CreatePolicy != "" {
		deployArgs = append(deployArgs, "--create-policy", p.CreatePolicy)
	}

	if p.DeletePolicy != "" {
		deployArgs = append(deployArgs, "--delete-policy", p.DeletePolicy)
	}

	if p.Description != "" {
		deployArgs = append(
			deployArgs, 
			fmt.Sprintf("--description=%s", p.Description),
		)
	}

	cmd := exec.Command(p.GcloudCmd, deployArgs...)
	trace(cmd)
	if ! p.Dryrun {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()

		if err != nil {
			return fmt.Errorf("Unable to update deployment: %s", err)
		}
	}

	return nil
}

func checkAndFixParams(p Plugin) (Plugin, error) {
	// check required params and set defaults
	p.Token = strings.TrimSpace(p.Token)
	if p.Token == "" {
		return p, fmt.Errorf("Missing required param: token")
	}

	// some of this behavior is borrowed from drone-gke
	// it will look at the token for project value
	// but maybe we would rather not make that a default
	// imagine a service account that belongs to a project,
	// but given IAM access to do deployments on other projects
	p.Project = strings.TrimSpace(p.Project)
	if p.Project == "" {
		p.Project = getProjectFromToken(p.Token)
		if p.Project == "" {
			return p, fmt.Errorf("Missing required param: project")
		}
	}

	if p.Deployment == "" {
		return p, fmt.Errorf("Missing required param: deployment")
	}

	sdkPath := "/google-cloud-sdk"

	// Defaults.
	p.GcloudCmd = strings.TrimSpace(p.GcloudCmd)
	if p.GcloudCmd == "" {
		p.GcloudCmd = fmt.Sprintf("%s/bin/gcloud", sdkPath)
	}

	p.ConfigTemplate = strings.TrimSpace(p.ConfigTemplate)
	if p.ConfigTemplate == "" {
		p.ConfigTemplate = ".gdm.yml"
	}

	return p, nil
}

func activateServiceAccount(p Plugin) error {
	keyPath := "/tmp/gcloud.json"

	// Write credentials to tmp file to be picked up by the 'gcloud' command.
	// This is inside the ephemeral plugin container, not on the host.
	err := ioutil.WriteFile(keyPath, []byte(p.Token), 0600)
	if err != nil {
		return fmt.Errorf("Error writing token file: %s\n", err)
	}

	// Warn if the keyfile can't be deleted, but don't abort.
	// We're almost certainly running inside an ephemeral container, so the file will be discarded anyway.
	defer func() {
		err := os.Remove(keyPath)
		if err != nil {
			fmt.Printf("Warning: error removing token file: %s\n", err)
		}
	}()

	cmd := exec.Command(
		p.GcloudCmd, 
		"auth",
		"activate-service-account",
		"--key-file",
		keyPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	trace(cmd)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to activate service account: %s", err)
	}

	return nil
}

func interpolateTemplate(p Plugin) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Error while getting working directory: %s\n", err)
	}

	inPath := filepath.Join(wd, p.ConfigTemplate)
	bn := filepath.Base(inPath)

	_, err = os.Stat(inPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("Error finding template: %s\n", err)
	}
	// Generate the file.
	blob, err := ioutil.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("Error reading template: %s\n", err)
	}

	tmpl, err := template.New(bn).Option("missingkey=error").Parse(string(blob))
	if err != nil {
		return fmt.Errorf("Error parsing template: %s\n", err)
	}

	f, err := os.Create(p.OutputFile)
	if err != nil {
		return fmt.Errorf("Error creating deployment config file: %s\n", err)
	}

	err = tmpl.Execute(f, p.Vars)
	if err != nil {
		return fmt.Errorf("Error executing deployment template: %s\n", err)
	}

	f.Close()

	return nil
}

// trace writes each command to stdout with the command wrapped in an xml
// tag so that it can be extracted and displayed in the logs.
func trace(cmd *exec.Cmd) {
	fmt.Fprintf(os.Stdout, "+ %s\n", strings.Join(cmd.Args, " "))
}

type token struct {
	ProjectID string `json:"project_id"`
}

func getProjectFromToken(j string) string {
	t := token{}
	err := json.Unmarshal([]byte(j), &t)
	if err != nil {
		return ""
	}
	return t.ProjectID
}

func getNewPath(path string) (string, error) {
	for i := 0; i <= 100; i++ {
		newPath := fmt.Sprintf("%s.%d", path, i)
		_, err := os.Stat(newPath)
		if os.IsNotExist(err) {
			return newPath, nil
		}
	}
	return "", fmt.Errorf("Unable to getNewPath, all permutations existed already")
}

func dumpData(w io.Writer, caption string, data interface{}) {
	fmt.Fprintf(w, "---START %s---\n", caption)
	defer fmt.Fprintf(w, "---END %s---\n", caption)

	b, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		fmt.Fprintf(w, "error marshalling: %s\n", err)
		return
	}

	w.Write(b)
	fmt.Fprintf(w, "\n")
}

func dumpFile(w io.Writer, caption, path string) {
	fmt.Fprintf(w, "---START %s---\n", caption)
	defer fmt.Fprintf(w, "---END %s---\n", caption)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Fprintf(w, "error reading file: %s\n", err)
		return
	}

	fmt.Fprintln(w, string(data))
}