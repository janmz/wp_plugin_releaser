package main

/*
 * wp_plugin_release: A tool to update plugins on your own server.
 *
 * It evaluates the file update_info.json from an updates directory and adjusts
 * the main file accordingly, reads the version from there, modifies update_info.json,
 * creates the ZIP file, makes it available under Updates, and then places both files
 * on the web server.
 *
 * Dependencies:
 * sconfig.go: Reading the config file with secure passwords
 * i18n.go: Internationalization of outputs and error messages
 *
 * Version: 1.3.1.65 (in version.go zu ändern)
 *
 * Author: Jan Neuhaus, VAYA Consulting, https://vaya-consultig.de/development/ https://github.com/janmz
 *
 * Repository: https://github.com/janmz/wp_plugin_releaser
 *
 * ChangeLog:
 *  28.04.26	1.3.1	Fix: existing version information not removed, but extended
 *  15.04.26	1.3.0	Feature: Include last five changes in the update_info.json
 *  15.04.26	1.2.15	Fix: changelog writes now always include a blank line at the end
 *  15.04.26	1.2.14	Fix: -trustserver works with exisiting host_key file and -c works also for the changelog message
 *  15.04.26	1.2.13	Feature: accept host key with -trustserver, allow -c oder -commit to give a commit message
 *  15.04.26	1.2.12	Fix: really make sure, PNG are rebuild if SVGs are updated
 *  21.02.26	1.2.11	Fixed host key verification (ssh_known_hosts, ~/.ssh/known_hosts), sftp, path checks
 *  12.02.26	1.2.10	Fixed: build-release process changed to clone the required janmz/sconfig
 *  13.12.25	1.2.9	Disabled debug output from sconfig
 *  03.12.25	1.2.8	fix: using debug version of sconfig
 *  02.12.25	1.2.6	fix: using newst version of sconfig
 *  20.11.25	1.2.5	fix: build and release workflow and cmd.Run() without error check
 *  06.11.25	1.2.4	fixed missing sync/push after commit
 *  06.11.25	1.2.3	fixed search for SVG tools, always compare to HEAD to find changed files
 *  06.11.25	1.2.2	fixed regexp for changelog parsing, fixed some lint errors
 *  06.11.25	1.2.1	fixed regexp for changelog parsing
 *  01.11.25	1.2.0	github integration, building of png from svg, check before upload
 *  01.11.25  	1.2.0	GitHub integration added
 *  17.08.25  	1.1.3	Internationalization and changelog added
 *  12.08.25  	1.1.0	Provided via GitHub
 *  08.08.25  	1.0.0	First version created and tested
 *
 * (c)2025 Jan Neuhaus, VAYA Consulting
 *
 */

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/janmz/sconfig"
)

var logger *log.Logger
var logFile *os.File
var config ConfigType

func parseCLIArgs(args []string) (workDir string, trustServer bool, commitMessage string) {
	workDir = ""
	trustServer = false
	commitMessage = ""

	for i := 0; i < len(args); i++ {
		a := strings.TrimSpace(args[i])
		if a == "" {
			continue
		}
		switch a {
		case "-trustserver":
			trustServer = true
			continue
		case "-c", "-commit":
			if i+1 < len(args) {
				commitMessage = strings.TrimSpace(args[i+1])
				i++
			}
			continue
		default:
			if strings.HasPrefix(a, "-") {
				continue
			}
			if workDir == "" {
				workDir = a
			}
		}
	}
	return workDir, trustServer, commitMessage
}

func main() {
	executablePath, err := os.Executable()
	if err != nil {
		executablePath = os.Args[0]
	}

	var buildTimeStr string
	buildTime, err := time.Parse("2006-01-02 15:04:05", BuildTime)
	if err != nil {
		buildTimeStr = BuildTime
	} else {
		buildTimeStr = buildTime.Local().Format("2006-01-02 15:04:05")
	}

	fmt.Printf("%s, %s\n", t("app.executable_path", executablePath), t("app.version", Version, buildTimeStr))

	workDir, trustServer, commitMessage := parseCLIArgs(os.Args[1:])

	if workDir == "" {
		var err2 error
		workDir, err2 = os.Getwd()
		if err2 != nil {
			fmt.Printf("%s", t("error.current_directory", err2)+"\n")
			os.Exit(1)
		}
	}
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		fmt.Printf("%s", t("error.no_directory", workDir)+"\n")
		os.Exit(1)
	}

	updateConfigPath := filepath.Join(workDir, "update.config")
	if _, err := os.Stat(updateConfigPath); os.IsNotExist(err) {
		fmt.Printf("%s", t("error.no_config", workDir)+"\n")
		os.Exit(1)
	}

	initLogging(workDir)
	defer logFile.Close()

	err = sconfig.LoadConfig(&config, 2, updateConfigPath, false, false)
	if err != nil {
		logAndPrint(t("error.config_read", err))
		os.Exit(1)
	}

	updateInfoPath := filepath.Join(workDir, "Updates", "update_info.json")
	updateInfo, allData, err := getUpdateInfo(updateInfoPath)
	if err != nil {
		logAndPrint(t("error.update_info_read", err))
		os.Exit(1)
	}

	logAndPrint(t("app.working_directory", workDir))

	currentVersion, err := processMainPHPFile(workDir, config.MainPHPFile, updateInfo)
	if err != nil {
		logAndPrint(t("error.php_processing", err))
		os.Exit(1)
	}
	logAndPrint(t("log.current_version_detected", currentVersion))

	err = processUpdateInfo(updateInfo, currentVersion)
	if err != nil {
		logAndPrint(t("error.update_info_processing", err))
		os.Exit(1)
	}

	changelogText, err := processChangelog(workDir, currentVersion, updateInfo, commitMessage)
	if err != nil {
		logAndPrint(t("error.changelog_write", err))
	} else if changelogText != "" {
		updateChangelogInUpdateInfo(workDir, updateInfo, changelogText)
	}

	err = processSVGFiles(workDir, updateInfo)
	if err != nil {
		logAndPrint(t("error.svg_convert", err))
	}

	remoteZIPName := filepath.Base(updateInfo.DownloadURL)
	re := regexp.MustCompile(`-v?[0-9.]*\.zip$`)
	remoteZIPName2 := re.ReplaceAllString(remoteZIPName, "")
	if updateInfo.Slug == "" {
		updateInfo.Slug = remoteZIPName2
	}
	if re.MatchString(remoteZIPName2) {
		logAndPrint(t("error.zip_version_remove"))
		remoteZIPName2 = strings.TrimSuffix(remoteZIPName2, ".zip")
	}

	zipFileName := fmt.Sprintf("%s-v%s.zip", remoteZIPName2, currentVersion)
	zipPath := filepath.Join(workDir, "Updates", zipFileName)
	err = createZipFile(workDir, zipPath, config.SkipPattern, updateInfo.Slug)
	if err != nil {
		logAndPrint(t("error.zip_creation", err))
		os.Exit(1)
	}
	updateInfo.DownloadURL = strings.TrimSuffix(updateInfo.DownloadURL, remoteZIPName) + zipFileName
	logAndPrint(t("log.download_url_set", redactSensitiveURL(updateInfo.DownloadURL)))

	err = setUpdateInfo(updateInfo, allData, updateInfoPath)
	if err != nil {
		logAndPrint(t("error.update_info_processing", err))
		os.Exit(1)
	}
	logAndPrint(t("log.zip_file_created", zipFileName))

	if config.SSHHost != "" && config.SSHUser != "" {
		err = uploadFiles(&config, zipPath, updateInfoPath, workDir, updateInfo, trustServer)
		if err != nil {
			logAndPrint(t("error.upload", err))
		} else {
			logAndPrint(t("log.upload_completed"))
		}
	} else {
		logAndPrint(t("log.no_ssh_config"))
	}

	err = handleGitHubIntegration(workDir, updateInfo, zipPath, commitMessage)
	if err != nil {
		logAndPrint(t("error.github_check", err))
	}

	logAndPrint(t("app.release_process_completed"))
}

func initLogging(workDir string) {
	logPath := filepath.Join(workDir, "update.log")
	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) // # nosec G304
	if err != nil {
		fmt.Printf("%s", t("error.log_file", err)+"\n")
		os.Exit(1)
	}
	logger = log.New(logFile, "", log.LstdFlags)
}

func logAndPrint(message string) {
	fmt.Println(message)
	logger.Println(message)
}
