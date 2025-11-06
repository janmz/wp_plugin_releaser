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
 * Version: 1.2.1.30 (in version.go zu ändern)
 *
 * Author: Jan Neuhaus, VAYA Consulting, https://vaya-consultig.de/development/ https://github.com/janmz
 *
 * Repository: https://github.com/janmz/wp_plugin_releaser
 *
 * ChangeLog:
 *  06.11.25	1.2.1	fixed regexp for changelog parsing
 *  01.11.25	1.2.0	github integration, building of png from svg, check before upload
 * 01.11.2025  	1.2.0	GitHub integration added
 * 17.08.2025  	1.1.3	Internationalization and changelog added
 * 12.08.2025  	1.1.0	Provided via GitHub
 * 08.08.2025  	1.0.0	First version created and tested
 *
 * (c)2025 Jan Neuhaus, VAYA Consulting
 *
 */

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/janmz/sconfig"
	"golang.org/x/crypto/ssh"
)

// ConfigType structure for update.config
type ConfigType struct {
	Version           int      `json:"version" default:"0"`
	MainPHPFile       string   `json:"main_php_file"`
	SkipPattern       []string `json:"skip_pattern"`
	SSHHost           string   `json:"ssh_host"`
	SSHPort           string   `json:"ssh_port"`
	SSHDirBase        string   `json:"ssh_dir_base"`
	SSHUser           string   `json:"ssh_user"`
	SSHKeyFile        string   `json:"ssh_key_file"`
	SSHPassword       string   `json:"ssh_password"`
	SSHSecurePassword string   `json:"ssh_secure_password"`
}

// UpdateInfo structure for update_info.json
type UpdateInfo struct {
	Version         string                 `json:"version"`
	LastUpdated     string                 `json:"last_updated"`
	DownloadURL     string                 `json:"download_url"`
	Details         string                 `json:"details,omitempty"`
	DetailsURL      string                 `json:"details_url,omitempty"`
	Upgrade_notice  string                 `json:"upgrade_notice,omitempty"`
	TestedWP        string                 `json:"tested,omitempty"`
	RequiresWP      string                 `json:"requires,omitempty"`
	RequiresPHP     string                 `json:"requires_php,omitempty"`
	Homepage        string                 `json:"homepage,omitempty"`
	Sections        map[string]string      `json:"sections,omitempty"`
	Banners         map[string]string      `json:"banners,omitempty"`
	Icons           map[string]string      `json:"icons,omitempty"`
	Screenshots     []map[string]string    `json:"screenshots,omitempty"`
	Contributors    map[string]string      `json:"contributors,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
	Donate_link     string                 `json:"donate_link,omitempty"`
	Ratings         map[string]int         `json:"ratings,omitempty"`
	Rating          float64                `json:"rating,omitempty"`
	NumRatings      int                    `json:"num_ratings,omitempty"`
	Downloaded      int                    `json:"downloaded,omitempty"`
	Active_installs int                    `json:"active_installs,omitempty"`
	Added           string                 `json:"added,omitempty"`
	Slug            string                 `json:"slug,omitempty"`
	Name            string                 `json:"name,omitempty"`
	Author          string                 `json:"author,omitempty"`
	AuthorProfile   string                 `json:"author_homepage,omitempty"`
	Extra           map[string]interface{} `json:"-"`
}

var logger *log.Logger
var logFile *os.File
var config ConfigType

func main() {

	// Determine working directory
	var workDir string
	if len(os.Args) > 1 {
		workDir = os.Args[1]
	} else {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			fmt.Printf(t("error.current_directory", err) + "\n")
			os.Exit(1)
		}
	}
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		fmt.Printf(t("error.no_directory", workDir) + "\n")
		os.Exit(1)
	}
	updateConfigPath := filepath.Join(workDir, "update.config")
	if _, err := os.Stat(updateConfigPath); os.IsNotExist(err) {
		fmt.Printf(t("error.no_config", workDir) + "\n")
		os.Exit(1)
	}
	// Initialize logging
	initLogging(workDir)
	defer logFile.Close()

	// Read config file
	err := sconfig.LoadConfig(&config, 2, updateConfigPath, false)
	if err != nil {
		logAndPrint(t("error.config_read", err))
		os.Exit(1)
	}

	// Read update_info.json file
	updateInfoPath := filepath.Join(workDir, "Updates", "update_info.json")
	updateInfo, allData, err := getUpdateInfo(updateInfoPath)
	if err != nil {
		logAndPrint(t("error.update_info_read", err))
		os.Exit(1)
	}

	// Display application name and version
	appName := filepath.Base(os.Args[0])
	logAndPrint(appName + " " + t("app.version", Version, BuildTime) + " " + t("app.working_directory", workDir))

	// Read and update main PHP file
	currentVersion, err := processMainPHPFile(workDir, config.MainPHPFile, updateInfo)
	if err != nil {
		logAndPrint(t("error.php_processing", err))
		os.Exit(1)
	}

	logAndPrint(t("log.current_version_detected", currentVersion))

	// Update update_info.json if needed
	err = processUpdateInfo(updateInfo, currentVersion)
	if err != nil {
		logAndPrint(t("error.update_info_processing", err))
		os.Exit(1)
	}

	// Process changelog
	changelogText, err := processChangelog(workDir, currentVersion, updateInfo)
	if err != nil {
		logAndPrint(t("error.changelog_write", err))
		// Don't exit, just continue without changelog
	} else if changelogText != "" {
		updateChangelogInUpdateInfo(updateInfo, changelogText)
	}

	// Check and convert SVG files if changed
	err = processSVGFiles(workDir, updateInfo)
	if err != nil {
		logAndPrint(t("error.svg_convert", err))
		// Don't exit, just continue
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
	// Create ZIP file
	zipFileName := fmt.Sprintf("%s-v%s.zip", remoteZIPName2, currentVersion)
	zipPath := filepath.Join(workDir, "Updates", zipFileName)
	err = createZipFile(workDir, zipPath, config.SkipPattern, updateInfo.Slug)
	if err != nil {
		logAndPrint(t("error.zip_creation", err))
		os.Exit(1)
	}
	updateInfo.DownloadURL = strings.TrimSuffix(updateInfo.DownloadURL, remoteZIPName) + zipFileName
	logAndPrint(t("log.download_url_set", updateInfo.DownloadURL))

	err = setUpdateInfo(updateInfo, allData, updateInfoPath)
	if err != nil {
		logAndPrint(t("error.update_info_processing", err))
		os.Exit(1)
	}

	logAndPrint(t("log.zip_file_created", zipFileName))

	// Upload via SSH if configured
	if config.SSHHost != "" && config.SSHUser != "" {
		err = uploadFiles(&config, zipPath, updateInfoPath, workDir, updateInfo)
		if err != nil {
			logAndPrint(t("error.upload", err))
		} else {
			logAndPrint(t("log.upload_completed"))
		}
	} else {
		logAndPrint(t("log.no_ssh_config"))
	}

	err = handleGitHubIntegration(workDir, updateInfo, zipPath)
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
		fmt.Printf(t("error.log_file", err) + "\n")
		os.Exit(1)
	}
	logger = log.New(logFile, "", log.LstdFlags)
}

func logAndPrint(message string) {
	fmt.Println(message)
	logger.Println(message)
}

func processMainPHPFile(workDir, mainPHPFile string, updateInfo *UpdateInfo) (string, error) {
	phpFilePath := filepath.Join(workDir, mainPHPFile)
	logAndPrint(t("log.processing_php", phpFilePath))

	content, err := os.ReadFile(phpFilePath) // # nosec G304
	if err != nil {
		return "", fmt.Errorf("%s", t("error.php_read_file", err))
	}

	contentStr := string(content)

	// Extract version from plugin comment
	commentVersionRegex := regexp.MustCompile(`(?is)(?:/\*.*?\bVersion:\s*|//\s*Version:\s*)([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)
	commentMatch := commentVersionRegex.FindStringSubmatchIndex(contentStr)

	var commentVersion string
	if len(commentMatch) == 4 {
		commentVersion = contentStr[commentMatch[2]:commentMatch[3]]
		logAndPrint(t("log.version_comment_found", commentVersion))
	}

	// Extract version from class property
	classVersionRegex := regexp.MustCompile(`private\s+\$version\s*=\s*['"]+([0-9]+\.[0-9]+(?:\.[0-9]+)?)['"]+`)
	classMatch := classVersionRegex.FindStringSubmatchIndex(contentStr)

	var classVersion string
	if len(classMatch) == 4 {
		classVersion = contentStr[classMatch[2]:classMatch[3]]
		logAndPrint(t("log.version_class_found", classVersion))
	}

	// Extract version from define() statement
	defineVersionRegex := regexp.MustCompile(`define\s*\(\s*['"]([A-Z_]+)_VERSION['"]\s*,\s*['"]([0-9]+\.[0-9]+(?:\.[0-9]+)?)['"]\s*\)`)
	defineMatch := defineVersionRegex.FindStringSubmatchIndex(contentStr)

	var defineVersion string
	var defineKey string
	if len(defineMatch) >= 6 {
		defineKey = contentStr[defineMatch[2]:defineMatch[3]]
		defineVersion = contentStr[defineMatch[4]:defineMatch[5]]
		logAndPrint(t("log.version_define_found", defineKey+"_VERSION", defineVersion))
	}

	// Determine current version (higher of all three)
	currentVersion := getHigherVersion(commentVersion, classVersion)
	currentVersion = getHigherVersion(currentVersion, defineVersion)
	if currentVersion == "" {
		return "", fmt.Errorf("%s", t("error.no_valid_version"))
	} else {
		logAndPrint(t("log.update_info_version_updated", currentVersion))
	}

	// Update both versions to current version
	if classVersion != "" && classVersion != currentVersion {
		if len(classMatch) == 4 {
			contentStr = contentStr[:classMatch[2]] + currentVersion + contentStr[classMatch[3]:]
		}
		logAndPrint(t("log.version_class_updated", currentVersion))
	}

	if commentVersion != "" && commentVersion != currentVersion {
		if len(commentMatch) == 4 {
			contentStr = contentStr[:commentMatch[2]] + currentVersion + contentStr[commentMatch[3]:]
		}
		logAndPrint(t("log.version_comment_updated", currentVersion))
	}

	// Update define version if present
	if defineVersion != "" && defineVersion != currentVersion {
		if len(defineMatch) >= 6 {
			contentStr = contentStr[:defineMatch[4]] + currentVersion + contentStr[defineMatch[5]:]
			logAndPrint(t("log.version_define_updated", currentVersion))
		}
	}

	// Update Last-Update comment
	currentDate := time.Now().Format("2006-01-02 15:04:05")
	lastUpdateRegex := regexp.MustCompile(`(?is)(?:/\*.*?\bLast-Update:\s*|//\s*Last-Update:\s*)([0-9]{4}-[0-9]{2}-[0-9]{2}( [0-9]{2}:[0-9]{2}(:[0-9]{2})?)?)`)
	lastUpdateMatch := lastUpdateRegex.FindStringSubmatchIndex(contentStr)

	if len(lastUpdateMatch) >= 4 {
		contentStr = contentStr[:lastUpdateMatch[2]] + currentDate + contentStr[lastUpdateMatch[3]:]
		logAndPrint(t("log.last_update_updated", currentDate))
	} else {
		// Add Last-Update comment after Version line
		commentVersionRegex = regexp.MustCompile(`(?is)(?:/\*.*?|//\s*)(\bVersion:\s*[0-9]+\.[0-9]+(?:\.[0-9]+)?)`)
		commentMatch := commentVersionRegex.FindStringSubmatchIndex(contentStr)
		if len(commentMatch) == 4 {
			posBeforeVersion := commentMatch[2]
			for posBeforeVersion > 0 && contentStr[posBeforeVersion] != '\n' {
				posBeforeVersion--
			}
			contentStr = contentStr[:commentMatch[1]] + contentStr[posBeforeVersion:commentMatch[2]] + fmt.Sprintf("Last-Update: %s", currentDate) + contentStr[commentMatch[1]:]
			logAndPrint(t("log.last_update_added", currentDate))
		}
	}
	// Check the Integration of PluginUpdateChecker
	pucRegex := regexp.MustCompile(`(?s)\$?[a-zA-Z0-9_]*::buildUpdateChecker\(\s*'([^']*)'\s*,\s*__FILE__,\s*(//[^\n]*)?\s*'([-_a-zA-Z0-9]*)'\s*\)`)
	pucMatch := pucRegex.FindStringSubmatchIndex(contentStr)
	newDownloadURL := strings.Replace(updateInfo.DownloadURL, filepath.Base(updateInfo.DownloadURL), "update_info.json", 1)
	if len(pucMatch) != 8 {
		return "", fmt.Errorf("%s", t("error.no_valid_puc", phpFilePath))
	} else {
		oldDownloadURL := ""
		oldSlug := ""
		if pucMatch[2] > 0 && (pucMatch[3] > pucMatch[2]) {
			oldDownloadURL = contentStr[pucMatch[2]:pucMatch[3]]
			logAndPrint(t("log.puc_download_url", oldDownloadURL))
		} else {
			return "", fmt.Errorf("%s", t("error.no_valid_puc", phpFilePath))
		}
		if pucMatch[4] > 0 && (pucMatch[5] > pucMatch[4]) {
			logAndPrint(t("log.puc_comment", contentStr[pucMatch[4]:pucMatch[5]]))
		}
		if pucMatch[6] > 0 && (pucMatch[7] > pucMatch[6]) {
			oldSlug = contentStr[pucMatch[6]:pucMatch[7]]
			logAndPrint(t("log.puc_slug", oldSlug))
		}
		needUpdate := false
		lengthdiff := 0
		newSlug := updateInfo.Slug
		if newSlug != oldSlug {
			contentStr = contentStr[:pucMatch[6]] + newSlug + contentStr[pucMatch[7]:]
			lengthdiff += len(newSlug) - len(oldSlug)
			needUpdate = true
		}
		if newDownloadURL != oldDownloadURL {
			contentStr = contentStr[:pucMatch[2]] + newDownloadURL + contentStr[pucMatch[3]:]
			lengthdiff += len(newDownloadURL) - len(oldDownloadURL)
			needUpdate = true
		}
		if needUpdate {
			logAndPrint(t("log.puc_changed", contentStr[pucMatch[0]:(pucMatch[1]+lengthdiff)]))
		}
	}

	// Write updated content back to file
	err = os.Rename(phpFilePath, phpFilePath+".bak")
	if err != nil {
		return "", fmt.Errorf(t("error.rename_file"), err)
	}
	err = os.WriteFile(phpFilePath, []byte(contentStr), 0600)
	if err != nil {
		return "", fmt.Errorf(t("error.write_php"), err)
	}

	logAndPrint(t("log.php_updated"))
	return currentVersion, nil
}

func getHigherVersion(v1, v2 string) string {
	if v1 == "" && v2 == "" {
		return ""
	}
	if v1 == "" {
		return v2
	}
	if v2 == "" {
		return v1
	}

	// Simple version comparison
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			n1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			n2, _ = strconv.Atoi(parts2[i])
		}

		if n1 > n2 {
			return v1
		}
		if n2 > n1 {
			return v2
		}
	}

	return v1 // Equal versions, return first
}

// marshalWithoutHTMLescaping schreibt JSON-Daten ohne HTML-Sonderzeichen zu ersetzen
// und sorgt für eine saubere Formatierung.
func marshalWithoutHTMLescaping(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false) // Dies ist die entscheidende Zeile!
	encoder.SetIndent("", "  ")  // Stellt die "schöne" Formatierung sicher
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Liest die Datei update_info.json ein und gibt die enthaltenen Informationen zurück.
// Dabei werden Informationen, die nicht in der bekannten Definition von UpdateInfo
// enthalten sind in einer map zwischengespeichert.
//
// Diese Funktion prüft auch, ob die Datei existiert und ob das JSON-Format korrekt ist.
//
// @param updateInfoPath Pfad zur update_info.json-Datei
// @return *UpdateInfo, map[string]interface{}, error
// @see UpdateInfo
func getUpdateInfo(updateInfoPath string) (*UpdateInfo, map[string]interface{}, error) {
	logAndPrint(t("log.reading_update_info", updateInfoPath))

	// Prüfen, ob die Datei existiert
	if _, err := os.Stat(updateInfoPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("%s", t("error.update_info_missing"))
	}

	data, err := os.ReadFile(updateInfoPath) // # nosec G304
	if err != nil {
		return nil, nil, fmt.Errorf("%s", t("error.update_info_read_file", err))
	}

	// 1. Alle Daten in eine Map einlesen, um unbekannte Felder zu erhalten.
	var allData map[string]interface{}
	if err := json.Unmarshal(data, &allData); err != nil {
		return nil, nil, fmt.Errorf("%s", t("error.update_info_invalid_json", err))
	}

	// 2. Die gleichen Daten in das Struct einlesen, um mit bekannten Feldern typsicher zu arbeiten.
	// Unbekannte Felder werden hierbei ignoriert.
	var updateInfo UpdateInfo
	if err := json.Unmarshal(data, &updateInfo); err != nil {
		return nil, nil, fmt.Errorf(t("error.update_info_structure"), err)
	}

	logAndPrint(t("log.current_version_update_info", updateInfo.Version))

	// Struct und die komplette Map zurückgeben
	return &updateInfo, allData, nil
}

func processUpdateInfo(updateInfo *UpdateInfo, currentVersion string) error {

	logAndPrint(t("log.processing_update_info"))

	// Check if update is needed
	if getHigherVersion(updateInfo.Version, currentVersion) == currentVersion && updateInfo.Version != currentVersion {
		updateInfo.Version = currentVersion
		updateInfo.LastUpdated = time.Now().Format("2006-01-02 15:04:05")
	}

	return nil
}

func setUpdateInfo(updateInfo *UpdateInfo, allData map[string]interface{}, updateInfoPath string) error {
	// Die modifizierten Daten aus dem Struct wieder in eine Map umwandeln.
	// Dies nutzt die eingebaute Logik von Go für `json`-Tags und `omitempty`.
	var structAsMap map[string]interface{}
	tempData, err := json.Marshal(updateInfo)
	if err != nil {
		return fmt.Errorf(t("error.json_prepare"), err)
	}
	if err := json.Unmarshal(tempData, &structAsMap); err != nil {
		return fmt.Errorf(t("error.json_mix"), err)
	}

	// Die aktualisierten Werte aus dem Struct in die Map mit allen Daten mischen.
	// Dies stellt sicher, dass unbekannte Felder erhalten bleiben.
	for key, value := range structAsMap {
		allData[key] = value
	}

	// Die finale Map mit unserer neuen Funktion ohne HTML-Escaping schreiben.
	updatedData, err := marshalWithoutHTMLescaping(allData)
	if err != nil {
		return fmt.Errorf(t("error.json_final"), err)
	}

	// Backup der alten Datei erstellen
	backupFilePath := updateInfoPath + ".bak"
	if err := os.Rename(updateInfoPath, backupFilePath); err != nil {
		return fmt.Errorf(t("error.backup_create"), err)
	}
	logAndPrint(t("log.update_info_backup", backupFilePath))

	// Neue Datei schreiben
	if err := os.WriteFile(updateInfoPath, updatedData, 0600); err != nil {
		return fmt.Errorf("%s", t("error.update_info_write_file", err))
	}

	logAndPrint(t("log.update_info_updated", updateInfo.Version))

	return nil
}

func createZipFile(sourceDir, zipPath string, skipPatterns []string, slug string) error {

	logAndPrint(t("log.creating_zip", zipPath))

	zipFile, err := os.Create(zipPath) // # nosec G304
	if err != nil {
		return fmt.Errorf(t("error.zip_create"), err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Default skip patterns
	defaultSkipPatterns := []string{
		"Updates",
		"update.config",
		"update.log",
		"*.code-workspace",
		"*.bak",
		"composer.lock",
		"Thumbs.db",
	}

	allSkipPatterns := append(defaultSkipPatterns, skipPatterns...)
	logAndPrint(t("log.skip_patterns", allSkipPatterns))

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Check skip patterns
		if shouldSkip(relPath, allSkipPatterns) {
			if info.IsDir() {
				logAndPrint(t("log.skip_directory", relPath))
				return filepath.SkipDir
			}
			logAndPrint(t("log.skip_file", relPath))
			return nil
		}

		// Skip directories in ZIP (they'll be created automatically)
		if info.IsDir() {
			return nil
		}

		// Add file to ZIP
		fileInZip, err := zipWriter.Create(slug + "/" + filepath.ToSlash(relPath))
		if err != nil {
			return err
		}

		fileContent, err := os.Open(path) // # nosec G304
		if err != nil {
			return err
		}
		defer fileContent.Close()

		_, err = io.Copy(fileInZip, fileContent)
		if err != nil {
			return err
		}

		logAndPrint(t("log.file_added", relPath))
		return nil
	})

	if err != nil {
		return fmt.Errorf(t("error.walk_files"), err)
	}

	logAndPrint(t("log.zip_created"))
	return nil
}

func shouldSkip(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}

		// Check if any part of the path matches
		parts := strings.Split(filepath.ToSlash(path), "/")
		for _, part := range parts {
			matched, err := filepath.Match(pattern, part)
			if err == nil && matched {
				return true
			}
		}
	}
	return false
}

func uploadFiles(config *ConfigType, zipPath, updateInfoPath string, workDir string, updateInfo *UpdateInfo) error {
	logAndPrint(t("log.ssh_upload_start"))

	// Setup authentication methods
	var authMethods []ssh.AuthMethod

	// Try SSH key authentication if key file is provided
	if config.SSHKeyFile != "" {
		key, err := os.ReadFile(config.SSHKeyFile)
		if err != nil {
			logAndPrint(t("log.ssh_key_warning", err))
		} else {
			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				logAndPrint(t("log.ssh_key_parse_warning", err))
			} else {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
				logAndPrint(t("log.ssh_key_added"))
			}
		}
	}

	// Add password authentication if password is provided
	if config.SSHPassword != "" {
		authMethods = append(authMethods, ssh.Password(config.SSHPassword))
		logAndPrint(t("log.ssh_password_added"))
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("%s", t("error.ssh_no_auth"))
	}

	// Setup SSH config ==> TODO include the HostKey-check and a workflow to get it!
	sshConfig := &ssh.ClientConfig{
		User:            config.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // #nosec G106
		Timeout:         30 * time.Second,
	}

	// Default port
	port := config.SSHPort
	if port == "" {
		port = "22"
	}

	// Connect to SSH server
	addr := fmt.Sprintf("%s:%s", config.SSHHost, port)
	logAndPrint(t("log.ssh_connecting", addr))

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf(t("error.ssh_connection"), err)
	}
	defer client.Close()
	logAndPrint(t("log.ssh_connected"))

	// Parse download URL to get remote path
	remoteLocalPath, err := parseRemotePath(updateInfo.DownloadURL, config.SSHDirBase)
	if err != nil {
		return err
	}

	logAndPrint(t("log.remote_path", remoteLocalPath))

	// Create remote directory if it doesn't exist
	err = createRemoteDir(client, remoteLocalPath)
	if err != nil {
		logAndPrint(t("log.remote_dir_warning", err))
	}
	// Upload ZIP file using SFTP
	err = uploadFileViaSFTP(client, zipPath, filepath.Join(remoteLocalPath, filepath.Base(zipPath)))
	if err != nil {
		return fmt.Errorf(t("error.zip_upload"), err)
	}

	// Upload update_info.json using SFTP
	err = uploadFileViaSFTP(client, updateInfoPath, filepath.Join(remoteLocalPath, "update_info.json"))
	if err != nil {
		return fmt.Errorf(t("error.update_info_upload"), err)
	}
	updatePath := filepath.Join(workDir, "Updates")
	if len(updateInfo.Banners) > 0 {
		for key, bannerUrl := range updateInfo.Banners {
			if _, err := url.Parse(bannerUrl); err == nil {
				bannerFilename := filepath.Base(bannerUrl)
				localBannerPath := filepath.Join(updatePath, bannerFilename)
				if _, err := os.Stat(localBannerPath); os.IsNotExist(err) {
					logAndPrint(t("log.banner_not_found", key, localBannerPath))
				} else {
					remoteBannerPath := filepath.Join(remoteLocalPath, bannerFilename)
					err = uploadFileViaSFTP(client, localBannerPath, remoteBannerPath)
					if err != nil {
						return fmt.Errorf(t("error.banner_upload"), err)
					}
				}
			} else {
				logAndPrint(t("log.banner_no_url", key, bannerUrl))
			}
		}
	}
	if len(updateInfo.Icons) > 0 {
		for key, iconUrl := range updateInfo.Icons {
			if _, err := url.Parse(iconUrl); err == nil {
				iconFilename := filepath.Base(iconUrl)
				localIconPath := filepath.Join(updatePath, iconFilename)
				if _, err := os.Stat(localIconPath); os.IsNotExist(err) {
					logAndPrint(t("log.icon_not_found", key, localIconPath))
				} else {
					remoteIconPath := filepath.Join(remoteLocalPath, iconFilename)
					err = uploadFileViaSFTP(client, localIconPath, remoteIconPath)
					if err != nil {
						return fmt.Errorf(t("error.icon_upload"), err)
					}
				}
			} else {
				logAndPrint(t("log.icon_no_url", key, iconUrl))
			}
		}
	}

	return nil
}

/*
 * Extrakting URL and Local path on server for given URL with filename
 */
func parseRemotePath(downloadURL string, basedir string) (string, error) {
	url_info, err := url.Parse(downloadURL)
	if err != nil {
		return "", err
	}
	path := url_info.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if strings.HasSuffix(path, "/") {
		return "", fmt.Errorf("%s", t("error.url_ends_directory", downloadURL))
	}
	pos := strings.LastIndex(path, "/")
	if pos < 0 {
		return "", fmt.Errorf("%s", t("error.url_no_filename", downloadURL))
	} else {
		path = path[:pos]
	}
	basedir = strings.TrimSuffix(basedir, "/")
	return basedir + path, nil
}

func createRemoteDir(client *ssh.Client, remotePath string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("mkdir -p %s", remotePath)
	err = session.Run(cmd)
	if err != nil {
		return err
	}

	logAndPrint(t("log.remote_dir_created", remotePath))
	return nil
}

func uploadFileViaSFTP(client *ssh.Client, localPath, remotePath string) error {
	remotePath = filepath.ToSlash(remotePath)
	logAndPrint(t("log.uploading_file", localPath, remotePath))

	// Check remote file modification time
	localInfo, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	remoteModTime, err := getRemoteFileModTime(client, remotePath)
	if err == nil {
		// Remote file exists, compare modification times
		if !localInfo.ModTime().After(remoteModTime) {
			// Local file is not newer, skip upload
			logAndPrint(t("log.file_already_current", filepath.Base(localPath)))
			return nil
		}
	}
	// Remote file doesn't exist or error occurred, proceed with upload

	// Create SFTP session
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Open local file
	localFile, err := os.Open(localPath) // # nosec G304
	if err != nil {
		return err
	}
	defer localFile.Close()

	// Create remote file using cat command
	remoteDir := filepath.Dir(remotePath)
	remoteDir = filepath.ToSlash(remoteDir)
	cmd := fmt.Sprintf("mkdir -p %s && cat > %s", remoteDir, remotePath)

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	// Start the remote command
	err = session.Start(cmd)
	if err != nil {
		return err
	}

	// Copy file content
	_, err = io.Copy(stdin, localFile)
	if err != nil {
		err2 := stdin.Close()
		if err2 != nil {
			return fmt.Errorf("failed to close stdin: %w; original error: %v", err2, err)
		}
		return err
	}

	// Close stdin to signal EOF
	err = stdin.Close()
	if err != nil {
		return err
	}

	// Wait for command to complete
	err = session.Wait()
	if err != nil {
		return err
	}

	logAndPrint(t("log.file_uploaded", filepath.Base(localPath)))
	return nil
}

// Changelog functions

// readChangelog reads existing changelog entries for a specific version
func readChangelog(workDir string, version string) (string, error) {
	changelogPath := filepath.Join(workDir, "CHANGELOG.md")
	if _, err := os.Stat(changelogPath); os.IsNotExist(err) {
		return "", nil
	}

	content, err := os.ReadFile(changelogPath)
	if err != nil {
		return "", err
	}

	contentStr := string(content)
	// Look for version section: ## [Version] or ## Version
	// Find the start of the version section (must be at start of line)
	versionPattern := fmt.Sprintf(`(?im)^##\s*\[?%s\]?`, regexp.QuoteMeta(version))
	versionStartRegex := regexp.MustCompile(versionPattern)
	startMatch := versionStartRegex.FindStringIndex(contentStr)
	if startMatch == nil {
		return "", nil
	}

	// Find the start of the next section (##) or end of string
	nextSectionRegex := regexp.MustCompile(`(?m)^##\s*\[?`)
	nextMatches := nextSectionRegex.FindAllStringIndex(contentStr, -1)

	var endPos int = len(contentStr)
	for _, match := range nextMatches {
		if match[0] > startMatch[0] {
			endPos = match[0]
			break
		}
	}

	// Extract the section content (skip the header line)
	sectionContent := contentStr[startMatch[0]:endPos]
	// Find the first newline after the header to get the actual content
	newlineIndex := strings.Index(sectionContent, "\n")
	if newlineIndex >= 0 {
		sectionContent = sectionContent[newlineIndex+1:]
	}

	return strings.TrimSpace(sectionContent), nil
}

// writeChangelog writes/updates changelog entries for a version
func writeChangelog(workDir string, version string, content string) error {
	changelogPath := filepath.Join(workDir, "CHANGELOG.md")
	currentDate := time.Now().Format("2006-01-02")

	var existingContent string
	var newContent string

	if _, err := os.Stat(changelogPath); os.IsNotExist(err) {
		// Create new changelog
		newContent = fmt.Sprintf("# Changelog\n\n## [%s] - %s\n\n%s\n", version, currentDate, content)
	} else {
		// Read existing content
		data, err := os.ReadFile(changelogPath)
		if err != nil {
			return err
		}
		existingContent = string(data)

		// Check if version entry already exists
		versionRegex := regexp.MustCompile(fmt.Sprintf(`(?is)(##\s*\[?%s\]?\s*-\s*[0-9-]+.*?\n)(.*?)(?=\n##\s*\[?|$)`, regexp.QuoteMeta(version)))
		if versionRegex.MatchString(existingContent) {
			// Replace existing entry
			newContent = versionRegex.ReplaceAllString(existingContent, fmt.Sprintf("## [%s] - %s\n\n%s\n", version, currentDate, content))
		} else {
			// Add new entry at the beginning (after # Changelog)
			changelogHeaderRegex := regexp.MustCompile(`(?is)^(#\s*Changelog\s*\n)`)
			if changelogHeaderRegex.MatchString(existingContent) {
				newContent = changelogHeaderRegex.ReplaceAllString(existingContent, fmt.Sprintf("$1\n## [%s] - %s\n\n%s\n\n", version, currentDate, content))
			} else {
				newContent = fmt.Sprintf("# Changelog\n\n## [%s] - %s\n\n%s\n\n%s", version, currentDate, content, existingContent)
			}
		}
	}

	// Write changelog
	return os.WriteFile(changelogPath, []byte(newContent), 0644)
}

// getChangedFiles detects changed files using git
func getChangedFiles(workDir string) ([]string, error) {
	// Check if .git exists
	if _, err := os.Stat(filepath.Join(workDir, ".git")); os.IsNotExist(err) {
		return []string{}, nil
	}

	// Try to get last tag
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	cmd.Dir = workDir
	lastTag, err := cmd.Output()
	if err != nil {
		// No tag found, compare against HEAD (staged and unstaged changes)
		cmd = exec.Command("git", "diff", "--name-only", "HEAD")
		cmd.Dir = workDir
		output, err := cmd.Output()
		if err != nil {
			return []string{}, nil // Ignore errors, return empty list
		}
		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		result := []string{}
		for _, file := range files {
			if file != "" {
				result = append(result, file)
			}
		}
		return result, nil
	}

	// Compare against last tag
	cmd = exec.Command("git", "diff", "--name-only", strings.TrimSpace(string(lastTag)), "HEAD")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return []string{}, nil // Ignore errors, return empty list
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := []string{}
	for _, file := range files {
		if file != "" {
			result = append(result, file)
		}
	}
	return result, nil
}

// isInteractiveTerminal checks if stdin is an interactive terminal
func isInteractiveTerminal() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Check if stdin is a character device (interactive terminal)
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// promptChangelogText prompts user for changelog text
func promptChangelogText(version string, existingText string, changedFiles []string) (string, error) {
	var preview strings.Builder

	if existingText != "" {
		preview.WriteString(existingText)
		preview.WriteString("\n\n")
	}

	if len(changedFiles) > 0 {
		preview.WriteString("Changed files:\n")
		for _, file := range changedFiles {
			preview.WriteString(fmt.Sprintf("- %s\n", file))
		}
	}

	if preview.Len() > 0 {
		fmt.Println(t("prompt.changelog_preview"))
		fmt.Println(preview.String())
	}

	// Check for environment variable to skip input (useful for debugging/testing)
	if os.Getenv("SKIP_CHANGELOG_INPUT") != "" || os.Getenv("AUTO_CHANGELOG") != "" {
		if preview.Len() > 0 {
			logAndPrint("Using auto-generated changelog (SKIP_CHANGELOG_INPUT or AUTO_CHANGELOG is set)")
			return strings.TrimSpace(preview.String()), nil
		}
		return "", nil
	}

	// Check if stdin is interactive (not available in debugger)
	if !isInteractiveTerminal() {
		if preview.Len() > 0 {
			logAndPrint("Non-interactive terminal detected, using auto-generated changelog")
			return strings.TrimSpace(preview.String()), nil
		}
		logAndPrint("Non-interactive terminal detected and no preview available, skipping changelog input")
		return "", nil
	}

	fmt.Print(t("prompt.changelog_text", version))
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		// If reading fails (e.g., in debugger), fall back to preview
		if preview.Len() > 0 {
			logAndPrint("Error reading input, using auto-generated changelog")
			return strings.TrimSpace(preview.String()), nil
		}
		return "", err
	}

	text := strings.TrimSpace(input)
	if text == "" && preview.Len() > 0 {
		// Use preview if user just presses Enter
		return strings.TrimSpace(preview.String()), nil
	}

	return text, nil
}

// processChangelog handles the complete changelog workflow
func processChangelog(workDir string, version string, updateInfo *UpdateInfo) (string, error) {
	logAndPrint(t("log.changelog_reading", version))

	// Read existing changelog entry
	existingText, err := readChangelog(workDir, version)
	if err != nil {
		logAndPrint(t("error.changelog_read", err))
	}

	// Get changed files
	changedFiles, err := getChangedFiles(workDir)
	if err != nil {
		logAndPrint(t("error.changed_files", err))
		changedFiles = []string{}
	} else {
		logAndPrint(t("log.changed_files_detected", len(changedFiles)))
	}

	// Prompt user for changelog text
	changelogText, err := promptChangelogText(version, existingText, changedFiles)
	if err != nil {
		return "", fmt.Errorf("%s", t("error.changelog_prompt", err))
	}

	if changelogText == "" {
		return "", nil
	}

	// Write changelog
	logAndPrint(t("log.changelog_writing", version))
	err = writeChangelog(workDir, version, changelogText)
	if err != nil {
		return "", err
	}
	logAndPrint(t("log.changelog_updated"))

	return changelogText, nil
}

// updateChangelogInUpdateInfo adds changelog to update_info.json as HTML
func updateChangelogInUpdateInfo(updateInfo *UpdateInfo, changelogText string) {
	if updateInfo.Sections == nil {
		updateInfo.Sections = make(map[string]string)
	}

	// Convert markdown to simple HTML (basic conversion)
	htmlText := html.EscapeString(changelogText)
	htmlText = strings.ReplaceAll(htmlText, "\n\n", "</p><p>")
	htmlText = strings.ReplaceAll(htmlText, "\n", "<br/>")
	htmlText = "<p>" + htmlText + "</p>"
	htmlText = regexp.MustCompile(`<p></p>`).ReplaceAllString(htmlText, "")
	htmlText = regexp.MustCompile(`- (.+)`).ReplaceAllString(htmlText, "<li>$1</li>")
	htmlText = strings.ReplaceAll(htmlText, "<p><li>", "<ul><li>")
	htmlText = strings.ReplaceAll(htmlText, "</li><br/>", "</li></ul><br/>")

	updateInfo.Sections["changelog"] = htmlText
	logAndPrint(t("log.changelog_in_update_info"))
}

// SVG conversion functions

// findSVGFiles finds all SVG files in the Updates directory
func findSVGFiles(updatesDir string) ([]string, error) {
	files, err := os.ReadDir(updatesDir)
	if err != nil {
		return nil, err
	}

	var svgFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".svg") {
			svgFiles = append(svgFiles, file.Name())
		}
	}

	return svgFiles, nil
}

// checkSVGFilesChanged checks if any SVG files have been modified
func checkSVGFilesChanged(workDir string) ([]string, error) {
	updatesDir := filepath.Join(workDir, "Updates")

	// Check via git if files changed
	if _, err := os.Stat(filepath.Join(workDir, ".git")); err == nil {
		changedFiles, err := getChangedFiles(workDir)
		if err == nil {
			var changedSVGFiles []string
			for _, file := range changedFiles {
				if strings.HasSuffix(strings.ToLower(file), ".svg") {
					// Get filename only
					filename := filepath.Base(file)
					changedSVGFiles = append(changedSVGFiles, filename)
				}
			}
			return changedSVGFiles, nil
		}
	}

	// If git check fails or no git repo, check all SVG files in Updates directory
	svgFiles, err := findSVGFiles(updatesDir)
	if err != nil {
		return nil, err
	}

	return svgFiles, nil
}

// convertSVGToPNG converts SVG files to PNG using external tool
func convertSVGToPNG(updatesDir string, svgFiles []string) error {
	// Check for available converter
	hasInkscape := false
	hasImageMagick := false

	if _, err := exec.LookPath("inkscape"); err == nil {
		hasInkscape = true
	}

	if _, err := exec.LookPath("convert"); err == nil {
		hasImageMagick = true
	}

	if !hasInkscape && !hasImageMagick {
		logAndPrint(t("error.svg_converter_missing"))
		logAndPrint("Skipping SVG to PNG conversion. Please install ImageMagick (convert) or Inkscape.")
		return nil // Don't treat as error, just skip
	}

	// Determine which converter to use (prefer Inkscape as it's more reliable for SVG)
	var converter func(string, string, []int, []int) error
	if hasInkscape {
		converter = convertSingleSVGWithInkscape
	} else {
		converter = convertSingleSVGWithImageMagick
	}

	// Convert each SVG file
	for _, svgFile := range svgFiles {
		svgPath := filepath.Join(updatesDir, svgFile)

		// Determine output sizes based on filename patterns or use defaults
		// Default sizes: square images get [128, 256], wide images get [772x250, 1544x500]
		squareSizes := []int{128, 256}
		wideSizes := [][]int{{772, 250}, {1544, 500}}

		filename := strings.ToLower(filepath.Base(svgFile))

		// Check if it looks like a logo/icon (square) or banner (wide)
		isLikelyLogo := strings.Contains(filename, "logo") || strings.Contains(filename, "icon")
		isLikelyBanner := strings.Contains(filename, "banner")

		if isLikelyLogo {
			// Generate square PNGs
			for _, size := range squareSizes {
				err := converter(svgPath, updatesDir, []int{size, size}, nil)
				if err != nil {
					return err
				}
			}
		} else if isLikelyBanner {
			// Generate banner PNGs
			for _, dims := range wideSizes {
				err := converter(svgPath, updatesDir, dims, nil)
				if err != nil {
					return err
				}
			}
		} else {
			// Unknown type - generate both square and banner sizes
			for _, size := range squareSizes {
				err := converter(svgPath, updatesDir, []int{size, size}, nil)
				if err != nil {
					return err
				}
			}
			for _, dims := range wideSizes {
				err := converter(svgPath, updatesDir, dims, nil)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// convertSingleSVGWithImageMagick converts a single SVG file to PNG with ImageMagick
func convertSingleSVGWithImageMagick(svgPath string, outputDir string, squareSize []int, wideSize []int) error {
	baseName := strings.TrimSuffix(filepath.Base(svgPath), ".svg")
	baseName = strings.TrimSuffix(baseName, ".SVG")

	var outputPath string
	var resizeArg string

	if len(squareSize) == 2 {
		// Square image
		outputPath = filepath.Join(outputDir, fmt.Sprintf("%s-%dx%d.png", baseName, squareSize[0], squareSize[1]))
		resizeArg = fmt.Sprintf("%dx%d", squareSize[0], squareSize[1])
	} else if len(wideSize) == 2 {
		// Wide/banner image
		outputPath = filepath.Join(outputDir, fmt.Sprintf("%s-%dx%d.png", baseName, wideSize[0], wideSize[1]))
		resizeArg = fmt.Sprintf("%dx%d", wideSize[0], wideSize[1])
	} else {
		return fmt.Errorf("invalid size parameters")
	}

	cmd := exec.Command("convert", "-background", "transparent", "-resize", resizeArg, svgPath, outputPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to convert %s: %v", svgPath, err)
	}

	logAndPrint(fmt.Sprintf("Converted: %s -> %s", filepath.Base(svgPath), filepath.Base(outputPath)))
	return nil
}

// convertSingleSVGWithInkscape converts a single SVG file to PNG with Inkscape
func convertSingleSVGWithInkscape(svgPath string, outputDir string, squareSize []int, wideSize []int) error {
	baseName := strings.TrimSuffix(filepath.Base(svgPath), ".svg")
	baseName = strings.TrimSuffix(baseName, ".SVG")

	var outputPath string
	var width, height string

	if len(squareSize) == 2 {
		// Square image
		outputPath = filepath.Join(outputDir, fmt.Sprintf("%s-%dx%d.png", baseName, squareSize[0], squareSize[1]))
		width = strconv.Itoa(squareSize[0])
		height = strconv.Itoa(squareSize[1])
	} else if len(wideSize) == 2 {
		// Wide/banner image
		outputPath = filepath.Join(outputDir, fmt.Sprintf("%s-%dx%d.png", baseName, wideSize[0], wideSize[1]))
		width = strconv.Itoa(wideSize[0])
		height = strconv.Itoa(wideSize[1])
	} else {
		return fmt.Errorf("invalid size parameters")
	}

	cmd := exec.Command("inkscape", "--export-filename", outputPath, "--export-width", width, "--export-height", height, svgPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to convert %s: %v", svgPath, err)
	}

	logAndPrint(fmt.Sprintf("Converted: %s -> %s", filepath.Base(svgPath), filepath.Base(outputPath)))
	return nil
}

// processSVGFiles checks and converts SVG files
func processSVGFiles(workDir string, updateInfo *UpdateInfo) error {
	updatesDir := filepath.Join(workDir, "Updates")

	// Check if Updates directory exists
	if _, err := os.Stat(updatesDir); os.IsNotExist(err) {
		return nil // No Updates directory, skip SVG processing
	}

	// Find changed SVG files
	changedSVGFiles, err := checkSVGFilesChanged(workDir)
	if err != nil {
		return err
	}

	if len(changedSVGFiles) == 0 {
		return nil // No SVG files to process
	}

	logAndPrint(t("log.svg_converting"))
	logAndPrint(fmt.Sprintf("Found %d SVG file(s) to convert", len(changedSVGFiles)))

	err = convertSVGToPNG(updatesDir, changedSVGFiles)
	if err != nil {
		return err
	}

	logAndPrint(t("log.svg_converted"))
	return nil
}

// GitHub integration functions

// promptGitHubUpdate asks user if GitHub update should be performed
func promptGitHubUpdate() bool {
	// Check for environment variable to auto-approve
	if autoUpdate := os.Getenv("AUTO_GITHUB_UPDATE"); autoUpdate != "" {
		text := strings.ToLower(strings.TrimSpace(autoUpdate))
		if text == "yes" || text == "y" || text == "j" || text == "ja" {
			logAndPrint("Auto-approving GitHub update (AUTO_GITHUB_UPDATE is set)")
			return true
		}
	}

	// Check for environment variable to skip
	if os.Getenv("SKIP_GITHUB_UPDATE") != "" {
		logAndPrint("Skipping GitHub update (SKIP_GITHUB_UPDATE is set)")
		return false
	}

	// Check if stdin is interactive
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
	// Accept y, yes, j, ja
	return text == "y" || text == "yes" || text == "j" || text == "ja"
}

// handleGitHubIntegration handles GitHub commit, tag and push after successful upload
func handleGitHubIntegration(workDir string, updateInfo *UpdateInfo, zipPath string) error {
	// Check if it's a GitHub repository
	isGitHub, err := isGitHubRepository(workDir)
	if err != nil {
		return err
	}

	if !isGitHub {
		logAndPrint(t("log.github_no_repo"))
		return nil
	}

	logAndPrint(t("log.github_repo_detected"))

	// Ask user for confirmation
	if !promptGitHubUpdate() {
		logAndPrint("GitHub update skipped by user")
		return nil
	}

	// Get changelog text from update_info.json (remove HTML tags for commit message)
	changelogText := ""
	if updateInfo.Sections != nil {
		htmlChangelog := updateInfo.Sections["changelog"]
		if htmlChangelog != "" {
			// Simple HTML tag removal for commit message
			changelogText = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(htmlChangelog, "")
			changelogText = strings.TrimSpace(changelogText)
		}
	}

	// Extract version from updateInfo
	version := updateInfo.Version
	if version == "" {
		// Try to extract from filename
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
	err = gitCommitAndTag(workDir, version, changelogText)
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

// isGitHubRepository checks if the project is in a GitHub repository
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
	// Check for GitHub URLs
	githubRegex := regexp.MustCompile(`(?i)(github\.com|githubusercontent\.com)`)
	return githubRegex.MatchString(contentStr), nil
}

// checkGitTagExists checks if a Git tag exists
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

// gitCommitAndTag commits changes and creates/updates tag
func gitCommitAndTag(workDir string, version string, changelogText string) error {
	if changelogText == "" {
		changelogText = fmt.Sprintf("Release version %s", version)
	}

	// Stage all changes
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %v", t("error.git_commit"), err)
	}

	// Commit
	cmd = exec.Command("git", "commit", "-m", changelogText)
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		// Check if there are changes to commit
		cmd = exec.Command("git", "diff", "--cached", "--quiet")
		cmd.Dir = workDir
		if err2 := cmd.Run(); err2 != nil {
			// There are changes, so commit failed
			return fmt.Errorf("%s: %v", t("error.git_commit"), err)
		}
		// No changes to commit, that's okay
	}

	tagName := fmt.Sprintf("v%s", version)

	// Check if tag exists
	tagExists, err := checkGitTagExists(workDir, version)
	if err != nil {
		return err
	}

	if tagExists {
		// Delete existing tag
		cmd = exec.Command("git", "tag", "-d", tagName)
		cmd.Dir = workDir
		cmd.Run() // Ignore errors
	}

	// Create tag
	cmd = exec.Command("git", "tag", "-a", tagName, "-m", changelogText)
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %v", t("error.git_tag"), err)
	}

	if tagExists {
		// Push tag deletion first
		cmd = exec.Command("git", "push", "origin", ":refs/tags/"+tagName)
		cmd.Dir = workDir
		cmd.Run() // Ignore errors
	}

	// Push tag
	cmd = exec.Command("git", "push", "origin", tagName)
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %v", t("error.git_tag"), err)
	}

	return nil
}

// syncToRemote pushes commits and tags to remote
func syncToRemote(workDir string) error {
	// Push commits
	cmd := exec.Command("git", "push")
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %v", t("error.git_push"), err)
	}

	return nil
}

// Upload optimization: check remote file modification time
func getRemoteFileModTime(client *ssh.Client, remotePath string) (time.Time, error) {
	session, err := client.NewSession()
	if err != nil {
		return time.Time{}, err
	}
	defer session.Close()

	cmd := fmt.Sprintf("stat -c %%Y '%s' 2>/dev/null || stat -f %%m '%s' 2>/dev/null || echo", remotePath, remotePath)
	output, err := session.Output(cmd)
	if err != nil {
		return time.Time{}, err
	}

	timestampStr := strings.TrimSpace(string(output))
	if timestampStr == "" {
		return time.Time{}, fmt.Errorf("file does not exist")
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(timestamp, 0), nil
}
