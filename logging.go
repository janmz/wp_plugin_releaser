package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
)

var verbose bool

func logVerbose(message string) {
	if verbose && message != "" {
		logAndPrint(message)
	}
}

func logOpenedFile(path string) {
	if verbose {
		logVerbose(t("log.opening_file", path))
	}
}

func runSystemCommand(workDir, name string, args ...string) error {
	if verbose {
		logVerbose(t("log.exec_command", formatCommand(name, args...)))
	}
	cmd := exec.Command(name, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

func runGitCommand(workDir string, args ...string) error {
	return runSystemCommand(workDir, "git", args...)
}

func runGitCommandOutput(workDir string, args ...string) ([]byte, error) {
	if verbose {
		logVerbose(t("log.exec_command", formatCommand("git", args...)))
		cmd := exec.Command("git", args...)
		cmd.Dir = workDir
		var stdout bytes.Buffer
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return stdout.Bytes(), err
		}
		return stdout.Bytes(), nil
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	return cmd.Output()
}

func formatCommand(name string, args ...string) string {
	parts := append([]string{name}, args...)
	return strings.Join(parts, " ")
}
