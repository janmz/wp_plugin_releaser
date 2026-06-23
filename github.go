package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func promptGitHubUpdate() bool {
	if autoUpdate := os.Getenv("AUTO_GITHUB_UPDATE"); autoUpdate != "" {
		text := strings.ToLower(strings.TrimSpace(autoUpdate))
		if text == "yes" || text == "y" || text == "j" || text == "ja" {
			logVerbose("Auto-approving GitHub update (AUTO_GITHUB_UPDATE is set)")
			return true
		}
	}

	if os.Getenv("SKIP_GITHUB_UPDATE") != "" {
		logVerbose("Skipping GitHub update (SKIP_GITHUB_UPDATE is set)")
		return false
	}

	if !isInteractiveTerminal() {
		logVerbose("Non-interactive terminal detected, skipping GitHub update")
		return false
	}

	fmt.Print(t("prompt.github_update"))
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		logVerbose("Error reading input, skipping GitHub update")
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
		logVerbose(t("log.github_no_repo"))
		return nil
	}

	logVerbose(t("log.github_repo_detected"))

	if !promptGitHubUpdate() {
		logVerbose("GitHub update skipped by user")
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
		logVerbose("Could not determine version for GitHub update")
		return nil
	}

	tagExists, err := checkGitTagExists(workDir, version)
	if err != nil {
		logAndPrint(t("error.git_tag_check", err))
		return err
	}
	if tagExists {
		logVerbose(t("log.git_tag_exists", version))
	} else {
		logVerbose(t("log.git_tag_not_exists", version))
	}

	logVerbose(t("log.git_committing"))
	err = gitCommitAndTag(workDir, version, changelogText, commitMessageOverride)
	if err != nil {
		logAndPrint(t("error.git_commit", err))
		return err
	}

	logVerbose(t("log.git_tagging", version))
	logVerbose(t("log.git_pushing"))
	err = syncToRemote(workDir)
	if err != nil {
		logAndPrint(t("error.git_push", err))
		return err
	}

	logVerbose(t("log.git_completed"))
	return nil
}

func isGitHubRepository(workDir string) (bool, error) {
	gitConfigPath := filepath.Join(workDir, ".git", "config")
	logOpenedFile(gitConfigPath)
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
	output, err := runGitCommandOutput(workDir, "tag", "-l", tagName)
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

	if err := runGitCommand(workDir, "add", "-A"); err != nil {
		return fmt.Errorf(t("error.git_commit"), err)
	}

	if err := runGitCommand(workDir, "commit", "-m", commitMessage); err != nil {
		if err2 := runGitCommand(workDir, "diff", "--cached", "--quiet"); err2 != nil {
			return fmt.Errorf(t("error.git_commit"), err)
		}
	}

	logVerbose(t("log.git_syncing"))
	if err := runGitCommand(workDir, "pull", "--rebase"); err != nil {
		if err2 := runGitCommand(workDir, "pull"); err2 != nil {
			return fmt.Errorf(t("error.git_sync"), err2)
		}
	}

	if err := runGitCommand(workDir, "push"); err != nil {
		return fmt.Errorf(t("error.git_push"), err)
	}

	tagName := fmt.Sprintf("v%s", version)

	tagExists, err := checkGitTagExists(workDir, version)
	if err != nil {
		return err
	}

	if tagExists {
		_ = runGitCommand(workDir, "tag", "-d", tagName) //nolint:errcheck
	}

	if err := runGitCommand(workDir, "tag", "-a", tagName, "-m", commitMessage); err != nil {
		return fmt.Errorf(t("error.git_tag"), err)
	}

	if tagExists {
		_ = runGitCommand(workDir, "push", "origin", ":refs/tags/"+tagName) //nolint:errcheck
	}

	if err := runGitCommand(workDir, "push", "origin", tagName); err != nil {
		return fmt.Errorf(t("error.git_tag"), err)
	}

	return nil
}

func syncToRemote(workDir string) error {
	if err := runGitCommand(workDir, "push"); err != nil {
		return fmt.Errorf(t("error.git_push"), err)
	}
	return nil
}
