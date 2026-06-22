// Command cicheck runs local CI checks (tests, i18n JSON, golangci-lint).
// Invoked by the pre-commit hook and `make ci`; works on Windows without bash.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	root, err := repoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := os.Chdir(root); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Running tests...")
	if err := run("go", "test", "./..."); err != nil {
		os.Exit(1)
	}

	fmt.Println("Validating translations...")
	for _, path := range []string{"locales/en.json", "locales/de.json"} {
		if err := validateJSON(path); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("Running golangci-lint...")
	if err := run(
		"go", "run",
		"github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8",
		"run", "--timeout=5m",
	); err != nil {
		os.Exit(1)
	}

	fmt.Println("CI checks passed.")
}

func repoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return os.Getwd()
	}
	return strings.TrimSpace(string(out)), nil
}

func validateJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("%s: invalid JSON: %w", path, err)
	}
	return nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
