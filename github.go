package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func promptGitHubUpdate() bool {
	if autoUpdate := os.Getenv("AUTO_GITHUB_UPDATE"); autoUpdate != "" {
		text := strings.ToLower(strings.TrimSpace(autoUpdate))
		if text == "yes" || text == "y" || text == "j" || text == "ja" {
			logAndPrint("Auto-approving GitHub update (AUTO_GITHUB_UPDATE is set)")
			return true
		}
	}

	if os.Getenv("SKIP_GITHUB_UPDATE") != "" {
		logAndPrint("Skipping GitHub update (SKIP_GITHUB_UPDATE is set)")
		return false
	}

	if !isInteractiveTerminal() {
		logAndPrint("Non-interactive terminal detected, skipping GitHub update")
		return false
	}

	fmt.Print(t("prompt.github_update"))
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		logAndPrint("Error reading input, skipping GitHub update")
		return false
	}

	text := strings.ToLower(strings.TrimSpace(input))
	return text == "y" || text == "yes" || text == "j" || text == "ja"
}

func handleGitHubIntegration(workDir string, updateInfo *UpdateInfo, zipPath string, commitMessageOverride string) error {
	isGitHub, err := isGitHubRepository(workDir)
	if err != nil {
		return err
	}
	if !isGitHub {
		logAndPrint(t("log.github_no_repo"))
		return nil
	}

	logAndPrint(t("log.github_repo_detected"))

	if !promptGitHubUpdate() {
		logAndPrint("GitHub update skipped by user")
		return nil
	}

	changelogText := ""
	if updateInfo.Sections != nil {
		htmlChangelog := updateInfo.Sections["changelog"]
		if htmlChangelog != "" {
			changelogText = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(htmlChangelog, "")
			changelogText = strings.TrimSpace(changelogText)
		}
	}

	version := updateInfo.Version
	if version == "" {
		re := regexp.MustCompile(`v?([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)
		if matches := re.FindStringSubmatch(filepath.Base(zipPath)); len(matches) > 1 {
			version = matches[1]
		}
	}
	if version == "" {
		logAndPrint("Could not determine version for GitHub update")
		return nil
	}

	tagExists, err := checkGitTagExists(workDir, version)
	if err != nil {
		logAndPrint(t("error.git_tag_check", err))
		return err
	}
	if tagExists {
		logAndPrint(t("log.git_tag_exists", version))
	} else {
		logAndPrint(t("log.git_tag_not_exists", version))
	}

	logAndPrint(t("log.git_committing"))
	err = gitCommitAndTag(workDir, version, changelogText, commitMessageOverride)
	if err != nil {
		logAndPrint(t("error.git_commit", err))
		return err
	}

	logAndPrint(t("log.git_tagging", version))
	logAndPrint(t("log.git_pushing"))
	err = syncToRemote(workDir)
	if err != nil {
		logAndPrint(t("error.git_push", err))
		return err
	}

	logAndPrint(t("log.git_completed"))
	return nil
}

func isGitHubRepository(workDir string) (bool, error) {
	gitConfigPath := filepath.Join(workDir, ".git", "config")
	if _, err := os.Stat(gitConfigPath); os.IsNotExist(err) {
		return false, nil
	}

	content, err := os.ReadFile(gitConfigPath)
	if err != nil {
		return false, err
	}

	contentStr := string(content)
	githubRegex := regexp.MustCompile(`(?i)(github\.com|githubusercontent\.com)`)
	return githubRegex.MatchString(contentStr), nil
}

func checkGitTagExists(workDir string, version string) (bool, error) {
	if _, err := os.Stat(filepath.Join(workDir, ".git")); os.IsNotExist(err) {
		return false, nil
	}

	tagName := fmt.Sprintf("v%s", version)
	cmd := exec.Command("git", "tag", "-l", tagName)
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(output)) == tagName, nil
}

func gitCommitAndTag(workDir string, version string, changelogText string, commitMessageOverride string) error {
	commitMessage := strings.TrimSpace(commitMessageOverride)
	if commitMessage == "" {
		commitMessage = changelogText
	}
	if commitMessage == "" {
		commitMessage = fmt.Sprintf("Release version %s", version)
	}

	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %v", t("error.git_commit"), err)
	}

	cmd = exec.Command("git", "commit", "-m", commitMessage)
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("git", "diff", "--cached", "--quiet")
		cmd.Dir = workDir
		if err2 := cmd.Run(); err2 != nil {
			return fmt.Errorf("%s: %v", t("error.git_commit"), err)
		}
	}

	logAndPrint(t("log.git_syncing"))
	cmd = exec.Command("git", "pull", "--rebase")
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("git", "pull")
		cmd.Dir = workDir
		if err2 := cmd.Run(); err2 != nil {
			return fmt.Errorf("%s: %v", t("error.git_sync"), err2)
		}
	}

	cmd = exec.Command("git", "push")
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %v", t("error.git_push"), err)
	}

	tagName := fmt.Sprintf("v%s", version)

	tagExists, err := checkGitTagExists(workDir, version)
	if err != nil {
		return err
	}

	if tagExists {
		cmd = exec.Command("git", "tag", "-d", tagName)
		cmd.Dir = workDir
		_ = cmd.Run() //nolint:errcheck
	}

	cmd = exec.Command("git", "tag", "-a", tagName, "-m", commitMessage)
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %v", t("error.git_tag"), err)
	}

	if tagExists {
		cmd = exec.Command("git", "push", "origin", ":refs/tags/"+tagName)
		cmd.Dir = workDir
		_ = cmd.Run() //nolint:errcheck
	}

	cmd = exec.Command("git", "push", "origin", tagName)
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %v", t("error.git_tag"), err)
	}

	return nil
}

func syncToRemote(workDir string) error {
	cmd := exec.Command("git", "push")
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %v", t("error.git_push"), err)
	}
	return nil
}

