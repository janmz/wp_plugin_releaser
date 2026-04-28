package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	SSHKnownHosts     string   `json:"ssh_known_hosts"`
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

// marshalWithoutHTMLescaping writes JSON without escaping HTML characters.
func marshalWithoutHTMLescaping(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// getUpdateInfo reads the update_info.json and preserves unknown fields.
func getUpdateInfo(updateInfoPath string) (*UpdateInfo, map[string]interface{}, error) {
	logAndPrint(t("log.reading_update_info", updateInfoPath))

	if _, err := os.Stat(updateInfoPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("%s", t("error.update_info_missing"))
	}

	data, err := os.ReadFile(updateInfoPath) // # nosec G304
	if err != nil {
		return nil, nil, fmt.Errorf("%s", t("error.update_info_read_file", err))
	}

	var allData map[string]interface{}
	if err := json.Unmarshal(data, &allData); err != nil {
		return nil, nil, fmt.Errorf("%s", t("error.update_info_invalid_json", err))
	}

	var updateInfo UpdateInfo
	if err := json.Unmarshal(data, &updateInfo); err != nil {
		return nil, nil, fmt.Errorf(t("error.update_info_structure"), err)
	}

	logAndPrint(t("log.current_version_update_info", updateInfo.Version))
	return &updateInfo, allData, nil
}

func processUpdateInfo(updateInfo *UpdateInfo, currentVersion string) error {
	logAndPrint(t("log.processing_update_info"))

	if getHigherVersion(updateInfo.Version, currentVersion) == currentVersion &&
		updateInfo.Version != currentVersion {
		updateInfo.Version = currentVersion
		updateInfo.LastUpdated = time.Now().Format("2006-01-02 15:04:05")
	}

	return nil
}

func setUpdateInfo(updateInfo *UpdateInfo, allData map[string]interface{}, updateInfoPath string) error {
	var structAsMap map[string]interface{}
	tempData, err := json.Marshal(updateInfo)
	if err != nil {
		return fmt.Errorf(t("error.json_prepare"), err)
	}
	if err := json.Unmarshal(tempData, &structAsMap); err != nil {
		return fmt.Errorf(t("error.json_mix"), err)
	}

	for key, value := range structAsMap {
		allData[key] = value
	}

	updatedData, err := marshalWithoutHTMLescaping(allData)
	if err != nil {
		return fmt.Errorf(t("error.json_final"), err)
	}

	backupFilePath := updateInfoPath + ".bak"
	if err := os.Rename(updateInfoPath, backupFilePath); err != nil {
		return fmt.Errorf(t("error.backup_create"), err)
	}
	logAndPrint(t("log.update_info_backup", backupFilePath))

	if err := os.WriteFile(updateInfoPath, updatedData, 0600); err != nil {
		return fmt.Errorf("%s", t("error.update_info_write_file", err))
	}

	logAndPrint(t("log.update_info_updated", updateInfo.Version))
	return nil
}

func createZipFile(sourceDir, zipPath string, skipPatterns []string, slug string) error {
	if err := validatePluginSlug(slug); err != nil {
		return err
	}

	logAndPrint(t("log.creating_zip", zipPath))

	zipFile, err := os.Create(zipPath) // # nosec G304
	if err != nil {
		return fmt.Errorf(t("error.zip_create"), err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

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

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		if shouldSkip(relPath, allSkipPatterns) {
			if info.IsDir() {
				logAndPrint(t("log.skip_directory", relPath))
				return filepath.SkipDir
			}
			logAndPrint(t("log.skip_file", relPath))
			return nil
		}

		if info.IsDir() {
			return nil
		}

		fileInZip, err := zipWriter.Create(slug + "/" + filepath.ToSlash(relPath))
		if err != nil {
			return err
		}

		fileContent, err := os.Open(path) // # nosec G304
		if err != nil {
			return err
		}
		defer fileContent.Close()

		if _, err := io.Copy(fileInZip, fileContent); err != nil {
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

func processMainPHPFile(workDir, mainPHPFile string, updateInfo *UpdateInfo) (string, error) {
	phpFilePath, err := safeJoinWithinBase(workDir, mainPHPFile)
	if err != nil {
		return "", err
	}
	logAndPrint(t("log.processing_php", phpFilePath))

	content, err := os.ReadFile(phpFilePath) // # nosec G304
	if err != nil {
		return "", fmt.Errorf("%s", t("error.php_read_file", err))
	}

	contentStr := string(content)

	commentVersionRegex := regexp.MustCompile(`(?is)(?:/\*.*?\bVersion:\s*|//\s*Version:\s*)([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)
	commentMatch := commentVersionRegex.FindStringSubmatchIndex(contentStr)

	var commentVersion string
	if len(commentMatch) == 4 {
		commentVersion = contentStr[commentMatch[2]:commentMatch[3]]
		logAndPrint(t("log.version_comment_found", commentVersion))
	}

	classVersionRegex := regexp.MustCompile(`private\s+\$version\s*=\s*['"]+([0-9]+\.[0-9]+(?:\.[0-9]+)?)['"]+`)
	classMatch := classVersionRegex.FindStringSubmatchIndex(contentStr)

	var classVersion string
	if len(classMatch) == 4 {
		classVersion = contentStr[classMatch[2]:classMatch[3]]
		logAndPrint(t("log.version_class_found", classVersion))
	}

	defineVersionRegex := regexp.MustCompile(`define\s*\(\s*['"]([A-Z_]+)_VERSION['"]\s*,\s*['"]([0-9]+\.[0-9]+(?:\.[0-9]+)?)['"]\s*\)`)
	defineMatch := defineVersionRegex.FindStringSubmatchIndex(contentStr)

	var defineVersion string
	var defineKey string
	if len(defineMatch) >= 6 {
		defineKey = contentStr[defineMatch[2]:defineMatch[3]]
		defineVersion = contentStr[defineMatch[4]:defineMatch[5]]
		logAndPrint(t("log.version_define_found", defineKey+"_VERSION", defineVersion))
	}

	currentVersion := getHigherVersion(commentVersion, classVersion)
	currentVersion = getHigherVersion(currentVersion, defineVersion)
	if currentVersion == "" {
		return "", fmt.Errorf("%s", t("error.no_valid_version"))
	}
	logAndPrint(t("log.update_info_version_updated", currentVersion))

	if classVersion != "" && classVersion != currentVersion && len(classMatch) == 4 {
		contentStr = contentStr[:classMatch[2]] + currentVersion + contentStr[classMatch[3]:]
		logAndPrint(t("log.version_class_updated", currentVersion))
	}

	if commentVersion != "" && commentVersion != currentVersion && len(commentMatch) == 4 {
		contentStr = contentStr[:commentMatch[2]] + currentVersion + contentStr[commentMatch[3]:]
		logAndPrint(t("log.version_comment_updated", currentVersion))
	}

	if defineVersion != "" && defineVersion != currentVersion && len(defineMatch) >= 6 {
		contentStr = contentStr[:defineMatch[4]] + currentVersion + contentStr[defineMatch[5]:]
		logAndPrint(t("log.version_define_updated", currentVersion))
	}

	currentDate := time.Now().Format("2006-01-02 15:04:05")
	lastUpdateRegex := regexp.MustCompile(`(?is)(?:/\*.*?\bLast-Update:\s*|//\s*Last-Update:\s*)([0-9]{4}-[0-9]{2}-[0-9]{2}( [0-9]{2}:[0-9]{2}(:[0-9]{2})?)?)`)
	lastUpdateMatch := lastUpdateRegex.FindStringSubmatchIndex(contentStr)

	if len(lastUpdateMatch) >= 4 {
		contentStr = contentStr[:lastUpdateMatch[2]] + currentDate + contentStr[lastUpdateMatch[3]:]
		logAndPrint(t("log.last_update_updated", currentDate))
	} else {
		commentVersionRegex = regexp.MustCompile(`(?is)(?:/\*.*?|//\s*)(\bVersion:\s*[0-9]+\.[0-9]+(?:\.[0-9]+)?)`)
		commentMatch := commentVersionRegex.FindStringSubmatchIndex(contentStr)
		if len(commentMatch) == 4 {
			posBeforeVersion := commentMatch[2]
			for posBeforeVersion > 0 && contentStr[posBeforeVersion] != '\n' {
				posBeforeVersion--
			}
			contentStr = contentStr[:commentMatch[1]] +
				contentStr[posBeforeVersion:commentMatch[2]] +
				fmt.Sprintf("Last-Update: %s", currentDate) +
				contentStr[commentMatch[1]:]
			logAndPrint(t("log.last_update_added", currentDate))
		}
	}

	pucRegex := regexp.MustCompile(`(?s)\$?[a-zA-Z0-9_]*::buildUpdateChecker\(\s*'([^']*)'\s*,\s*__FILE__,\s*(//[^\n]*)?\s*'([-_a-zA-Z0-9]*)'\s*\)`)
	pucMatch := pucRegex.FindStringSubmatchIndex(contentStr)
	newDownloadURL := strings.Replace(updateInfo.DownloadURL, filepath.Base(updateInfo.DownloadURL), "update_info.json", 1)
	if len(pucMatch) != 8 {
		return "", fmt.Errorf("%s", t("error.no_valid_puc", phpFilePath))
	}

	oldDownloadURL := ""
	oldSlug := ""
	if pucMatch[2] > 0 && (pucMatch[3] > pucMatch[2]) {
		oldDownloadURL = contentStr[pucMatch[2]:pucMatch[3]]
		logAndPrint(t("log.puc_download_url", redactSensitiveURL(oldDownloadURL)))
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

	if err := os.Rename(phpFilePath, phpFilePath+".bak"); err != nil {
		return "", fmt.Errorf(t("error.rename_file"), err)
	}
	if err := os.WriteFile(phpFilePath, []byte(contentStr), 0600); err != nil {
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

	return v1
}

func safeJoinWithinBase(baseDir, relativePath string) (string, error) {
	if strings.TrimSpace(relativePath) == "" {
		return "", fmt.Errorf("main_php_file must not be empty")
	}
	if filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("main_php_file must be a relative path")
	}

	candidate := filepath.Join(baseDir, relativePath)
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	relToBase, err := filepath.Rel(baseAbs, candidateAbs)
	if err != nil {
		return "", err
	}
	if relToBase == ".." || strings.HasPrefix(relToBase, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("main_php_file escapes working directory: %s", relativePath)
	}
	return candidateAbs, nil
}

func validatePluginSlug(slug string) error {
	if strings.TrimSpace(slug) == "" {
		return fmt.Errorf("slug must not be empty")
	}
	allowed := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !allowed.MatchString(slug) {
		return fmt.Errorf("invalid slug %q: only letters, numbers, underscore and dash are allowed", slug)
	}
	return nil
}

func redactSensitiveURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if parsedURL.RawQuery == "" {
		return rawURL
	}
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""
	return parsedURL.String()
}

