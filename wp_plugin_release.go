package main

/*
 * wp_plugin_release: Ein Werkzeug, um Updates für Plugins auf einem eigenen Server zu aktualisieren.
 *
 * Es wird die Datei update_info.json aus einem Updates-Verzeichnis ausgewertet und entsprechend
 * die Hauptdatei angepasst und dort die Version ausgelesen, die update_info.json angepasst, die
 * ZIP-Datei erstellt und unter Updates bereitgestellt und die beiden Dateien dann auf dem
 * Webserver abgelegt.
 *
 * Abhängigkeiten:
 * config.go: Einlesen der Config-Datei mit sicheren Passwörtern
 * hardware-id.go: Bestimmung einer eindeutigen System-ID für die Schlüssel von config.go
 *
 * Version: In Datei version.go!
 *
 * Autor: Jan Neuhaus, VAYA Consulting, https://vaya-consultig.de/development/ https://github.com/janmz
 *
 * Repository: Https://github.com/janmz/wp_plugin_releaser
 *
 * Change_log:
 * 12.8.2025	Über github bereitgestellt
 * 8.8.2025		Erste Version erstellt und getestet
 *
 * (c)2025 Jan Neuhaus, VAYA Consulting
 *
 */

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

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
			fmt.Printf("Fehler beim Ermitteln des aktuellen Verzeichnisses: %v\n", err)
			os.Exit(1)
		}
	}
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		fmt.Printf("Das Verzeichnis %s existiert nicht oder ist nicht lesbar\n", workDir)
		os.Exit(1)
	}
	updateConfigPath := filepath.Join(workDir, "update.config")
	if _, err := os.Stat(updateConfigPath); os.IsNotExist(err) {
		fmt.Printf("Das Verzeichnis %s enthält keine Datei update.config\n", workDir)
		os.Exit(1)
	}
	// Initialize logging
	initLogging(workDir)
	defer logFile.Close()

	logAndPrint("WordPress Plugin Release Tool Version " + Version + " vom " + BuildTime + " gestartet")
	logAndPrint(fmt.Sprintf("Arbeitsverzeichnis: %s", workDir))

	// Read config file
	err := loadConfig(&config, 2, updateConfigPath, false)
	if err != nil {
		logAndPrint(fmt.Sprintf("Fehler beim Lesen der Konfiguration: %v", err))
		os.Exit(1)
	}

	// Read update_info.json file
	updateInfoPath := filepath.Join(workDir, "Updates", "update_info.json")
	updateInfo, allData, err := getUpdateInfo(updateInfoPath)
	if err != nil {
		logAndPrint(fmt.Sprintf("Fehler beim Lesen der Update-Info: %v", err))
		os.Exit(1)
	}

	// Read and update main PHP file
	currentVersion, err := processMainPHPFile(workDir, config.MainPHPFile, updateInfo)
	if err != nil {
		logAndPrint(fmt.Sprintf("Fehler beim Verarbeiten der PHP-Datei: %v", err))
		os.Exit(1)
	}

	logAndPrint(fmt.Sprintf("Aktuelle Version ermittelt: %s", currentVersion))

	// Update update_info.json if needed
	err = processUpdateInfo(updateInfo, currentVersion)
	if err != nil {
		logAndPrint(fmt.Sprintf("Fehler beim Anpassen der Update-Info: %v", err))
		os.Exit(1)
	}
	remoteZIPName := filepath.Base(updateInfo.DownloadURL)
	re := regexp.MustCompile(`-v?[0-9.]*\.zip$`)
	remoteZIPName2 := re.ReplaceAllString(remoteZIPName, "")
	if updateInfo.Slug == "" {
		updateInfo.Slug = remoteZIPName2
	}
	if re.MatchString(remoteZIPName2) {
		logAndPrint("Fehler: Konnte Versionummer im Namen der ZIP-Datei nicht entfernen")
		remoteZIPName2 = strings.TrimSuffix(remoteZIPName2, ".zip")
	}
	// Create ZIP file
	zipFileName := fmt.Sprintf("%s-v%s.zip", remoteZIPName2, currentVersion)
	zipPath := filepath.Join(workDir, "Updates", zipFileName)
	err = createZipFile(workDir, zipPath, config.SkipPattern, updateInfo.Slug)
	if err != nil {
		logAndPrint(fmt.Sprintf("Fehler beim Erstellen der ZIP-Datei: %v", err))
		os.Exit(1)
	}
	updateInfo.DownloadURL = strings.TrimSuffix(updateInfo.DownloadURL, remoteZIPName) + zipFileName
	logAndPrint(fmt.Sprintf("Download URL auf '%s' gesetzt!", updateInfo.DownloadURL))

	err = setUpdateInfo(updateInfo, allData, updateInfoPath)
	if err != nil {
		logAndPrint(fmt.Sprintf("Fehler beim Schreiben der Update-Info: %v", err))
		os.Exit(1)
	}

	logAndPrint(fmt.Sprintf("ZIP-Datei erstellt: %s", zipFileName))

	// Upload via SSH if configured
	if config.SSHHost != "" && config.SSHUser != "" {
		err = uploadFiles(&config, zipPath, updateInfoPath, updateInfo)
		if err != nil {
			logAndPrint(fmt.Sprintf("Fehler beim Upload: %v", err))
		} else {
			logAndPrint("Upload erfolgreich abgeschlossen")
		}
	} else {
		logAndPrint("Keine SSH-Konfiguration gefunden, Upload übersprungen")
	}

	logAndPrint("Release-Prozess erfolgreich abgeschlossen")
}

func initLogging(workDir string) {
	logPath := filepath.Join(workDir, "update.log")
	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Fehler beim Öffnen der Log-Datei: %v\n", err)
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
	logAndPrint(fmt.Sprintf("Verarbeite PHP-Datei: %s", phpFilePath))

	content, err := os.ReadFile(phpFilePath)
	if err != nil {
		return "", fmt.Errorf("PHP-Datei konnte nicht gelesen werden: %v", err)
	}

	contentStr := string(content)

	// Extract version from plugin comment
	commentVersionRegex := regexp.MustCompile(`(?i)\*\s*Version:\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)
	commentMatch := commentVersionRegex.FindStringSubmatch(contentStr)

	var commentVersion string
	if len(commentMatch) > 1 {
		commentVersion = commentMatch[1]
		logAndPrint(fmt.Sprintf("Version im Plugin-Kommentar gefunden: %s", commentVersion))
	}

	// Extract version from class property
	classVersionRegex := regexp.MustCompile(`private\s+\$version\s*=\s*['"]+([0-9]+\.[0-9]+(?:\.[0-9]+)?)['"]+`)
	classMatch := classVersionRegex.FindStringSubmatch(contentStr)

	var classVersion string
	if len(classMatch) > 1 {
		classVersion = classMatch[1]
		logAndPrint(fmt.Sprintf("Version in Klassen-Property gefunden: %s", classVersion))
	}

	// Determine current version (higher of both)
	currentVersion := getHigherVersion(commentVersion, classVersion)
	if currentVersion == "" {
		//lint:ignore ST1005 German error message requires capitalization
		return "", fmt.Errorf("Keine gültige Version gefunden")
	} else {
		logAndPrint(fmt.Sprintf("update_info.json Version aktualisiert auf: %s", currentVersion))
	}

	// Update both versions to current version
	if commentVersion != "" && commentVersion != currentVersion {
		contentStr = commentVersionRegex.ReplaceAllString(contentStr, fmt.Sprintf("* Version: %s", currentVersion))
		logAndPrint(fmt.Sprintf("Plugin-Kommentar Version aktualisiert auf: %s", currentVersion))
	}

	if classVersion != "" && classVersion != currentVersion {
		contentStr = classVersionRegex.ReplaceAllString(contentStr, fmt.Sprintf("private $$version = '%s'", currentVersion))
		logAndPrint(fmt.Sprintf("Klassen-Property Version aktualisiert auf: %s", currentVersion))
	}

	// Update Last-Update comment
	currentDate := time.Now().Format("2006-01-02 15:04:05")
	lastUpdateRegex := regexp.MustCompile(`(?i)\*\s*Last-Update:\s*[0-9]{4}-[0-9]{2}-[0-9]{2}( [0-9]{2}:[0-9]{2}(:[0-9]{2})?)?`)

	if lastUpdateRegex.MatchString(contentStr) {
		contentStr = lastUpdateRegex.ReplaceAllString(contentStr, fmt.Sprintf("* Last-Update: %s", currentDate))
		logAndPrint(fmt.Sprintf("Last-Update Kommentar aktualisiert auf: %s", currentDate))
	} else {
		// Add Last-Update comment after Version line
		versionLineRegex := regexp.MustCompile(`(\*\s*Version:\s*[0-9]+\.[0-9]+(?:\.[0-9]+)?\s*\n)`)
		if versionLineRegex.MatchString(contentStr) {
			contentStr = versionLineRegex.ReplaceAllString(contentStr, fmt.Sprintf("$1 * Last-Update: %s\n", currentDate))
			logAndPrint(fmt.Sprintf("Last-Update Kommentar hinzugefügt: %s", currentDate))
		}
	}
	// Check the Integration of PluginUpdateChecker
	pucRegex := regexp.MustCompile(`(?s)PucFactory::buildUpdateChecker\(\s*'([^']*)'\s*,\s*__FILE__,\s*(//[^\n]*)?\s*'([-_a-zA-Z0-9]*)'\s*\)`)
	pucMatch := pucRegex.FindStringSubmatchIndex(contentStr)
	newDownloadURL := strings.Replace(updateInfo.DownloadURL, filepath.Base(updateInfo.DownloadURL), "update_info.json", 1)
	if len(pucMatch) != 8 {
		//lint:ignore ST1005 German error message requires capitalization
		return "", fmt.Errorf("Keine gültige Puc-Integration gefunden, bitte Datei %s überarbeiten", phpFilePath)
	} else {
		oldDownloadURL := ""
		oldSlug := ""
		if pucMatch[2] > 0 && (pucMatch[3] > pucMatch[2]) {
			oldDownloadURL = contentStr[pucMatch[2]:pucMatch[3]]
			logAndPrint(fmt.Sprintf("Im PUC-Aufruf DownloadURL gefunden: %s", oldDownloadURL))
		} else {
			//lint:ignore ST1005 German error message requires capitalization
			return "", fmt.Errorf("Keine gültige Puc-Integration gefunden, bitte Datei %s überarbeiten", phpFilePath)
		}
		if pucMatch[4] > 0 && (pucMatch[5] > pucMatch[4]) {
			logAndPrint(fmt.Sprintf("Im PUC-Aufruf Kommentar gefunden: %s", contentStr[pucMatch[4]:pucMatch[5]]))
		}
		if pucMatch[6] > 0 && (pucMatch[7] > pucMatch[6]) {
			oldSlug = contentStr[pucMatch[6]:pucMatch[7]]
			logAndPrint(fmt.Sprintf("Im PUC AUfruf Slug gefunden: %s", oldSlug))
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
			logAndPrint(fmt.Sprintf("Puc-Integration geändert zu: %s", contentStr[pucMatch[0]:(pucMatch[1]+lengthdiff)]))
		}
	}

	// Write updated content back to file
	err = os.Rename(phpFilePath, phpFilePath+".bak")
	if err != nil {
		return "", fmt.Errorf("alte Datei konnte nicht umbenannt werden: %v", err)
	}
	err = os.WriteFile(phpFilePath, []byte(contentStr), 0644)
	if err != nil {
		return "", fmt.Errorf("PHP-Datei konnte nicht geschrieben werden: %v", err)
	}

	logAndPrint("PHP-Datei erfolgreich aktualisiert")
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
	logAndPrint(fmt.Sprintf("Einlesen Update-Info: %s", updateInfoPath))

	// Prüfen, ob die Datei existiert
	if _, err := os.Stat(updateInfoPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("update_info.json ist nicht vorhanden")
	}

	data, err := os.ReadFile(updateInfoPath)
	if err != nil {
		return nil, nil, fmt.Errorf("update_info.json konnte nicht gelesen werden: %v", err)
	}

	// 1. Alle Daten in eine Map einlesen, um unbekannte Felder zu erhalten.
	var allData map[string]interface{}
	if err := json.Unmarshal(data, &allData); err != nil {
		return nil, nil, fmt.Errorf("update_info.json hat ungültiges JSON-Format: %v", err)
	}

	// 2. Die gleichen Daten in das Struct einlesen, um mit bekannten Feldern typsicher zu arbeiten.
	// Unbekannte Felder werden hierbei ignoriert.
	var updateInfo UpdateInfo
	if err := json.Unmarshal(data, &updateInfo); err != nil {
		//lint:ignore ST1005 German error message requires capitalization
		return nil, nil, fmt.Errorf("Struktur von update_info.json konnte nicht analysiert werden: %v", err)
	}

	logAndPrint(fmt.Sprintf("Aktuelle Version in update_info.json: %s", updateInfo.Version))

	// Struct und die komplette Map zurückgeben
	return &updateInfo, allData, nil
}

func processUpdateInfo(updateInfo *UpdateInfo, currentVersion string) error {

	logAndPrint("Verarbeite Update-Infos")

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
		//lint:ignore ST1005 German error message requires capitalization
		return fmt.Errorf("Fehler beim Vorbereiten der JSON-Daten aus dem Struct: %v", err)
	}
	if err := json.Unmarshal(tempData, &structAsMap); err != nil {
		//lint:ignore ST1005 German error message requires capitalization
		return fmt.Errorf("Fehler beim Mischen der JSON-Daten: %v", err)
	}

	// Die aktualisierten Werte aus dem Struct in die Map mit allen Daten mischen.
	// Dies stellt sicher, dass unbekannte Felder erhalten bleiben.
	for key, value := range structAsMap {
		allData[key] = value
	}

	// Die finale Map mit unserer neuen Funktion ohne HTML-Escaping schreiben.
	updatedData, err := marshalWithoutHTMLescaping(allData)
	if err != nil {
		//lint:ignore ST1005 German error message requires capitalization
		return fmt.Errorf("Fehler beim Erstellen der finalen JSON-Daten: %v", err)
	}

	// Backup der alten Datei erstellen
	backupFilePath := updateInfoPath + ".bak"
	if err := os.Rename(updateInfoPath, backupFilePath); err != nil {
		//lint:ignore ST1005 German error message requires capitalization
		return fmt.Errorf("Fehler beim Erstellen der Sicherung für update_info.json: %v", err)
	}
	logAndPrint(fmt.Sprintf("Sicherung von update_info.json erstellt: %s", backupFilePath))

	// Neue Datei schreiben
	if err := os.WriteFile(updateInfoPath, updatedData, 0644); err != nil {
		return fmt.Errorf("update_info.json konnte nicht geschrieben werden: %v", err)
	}

	logAndPrint(fmt.Sprintf("update_info.json aktualisiert auf Version %s", updateInfo.Version))

	return nil
}

func createZipFile(sourceDir, zipPath string, skipPatterns []string, slug string) error {

	logAndPrint(fmt.Sprintf("Erstelle ZIP-Datei: %s", zipPath))

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("ZIP-Datei konnte nicht erstellt werden: %v", err)
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
		"Thumbs.db",
	}

	allSkipPatterns := append(defaultSkipPatterns, skipPatterns...)
	logAndPrint(fmt.Sprintf("Skip-Patterns: %v", allSkipPatterns))

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
				logAndPrint(fmt.Sprintf("Überspringe Verzeichnis: %s", relPath))
				return filepath.SkipDir
			}
			logAndPrint(fmt.Sprintf("Überspringe Datei: %s", relPath))
			return nil
		}

		// Skip directories in ZIP (they'll be created automatically)
		if info.IsDir() {
			return nil
		}

		// Add file to ZIP
		fileInZip, err := zipWriter.Create(slug + "/" + relPath)
		if err != nil {
			return err
		}

		fileContent, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fileContent.Close()

		_, err = io.Copy(fileInZip, fileContent)
		if err != nil {
			return err
		}

		logAndPrint(fmt.Sprintf("Datei hinzugefügt: %s", relPath))
		return nil
	})

	if err != nil {
		//lint:ignore ST1005 German error message requires capitalization
		return fmt.Errorf("Fehler beim Durchlaufen der Dateien: %v", err)
	}

	logAndPrint("ZIP-Datei erfolgreich erstellt")
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

func uploadFiles(config *ConfigType, zipPath, updateInfoPath string, updateInfo *UpdateInfo) error {
	logAndPrint("Beginne SSH-Upload")

	// Setup authentication methods
	var authMethods []ssh.AuthMethod

	// Try SSH key authentication if key file is provided
	if config.SSHKeyFile != "" {
		key, err := os.ReadFile(config.SSHKeyFile)
		if err != nil {
			logAndPrint(fmt.Sprintf("Warnung: SSH-Schlüssel konnte nicht gelesen werden: %v", err))
		} else {
			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				logAndPrint(fmt.Sprintf("Warnung: SSH-Schlüssel konnte nicht geparst werden: %v", err))
			} else {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
				logAndPrint("SSH-Schlüssel-Authentifizierung hinzugefügt")
			}
		}
	}

	// Add password authentication if password is provided
	if config.SSHPassword != "" {
		authMethods = append(authMethods, ssh.Password(config.SSHPassword))
		logAndPrint("SSH-Passwort-Authentifizierung hinzugefügt")
	}

	if len(authMethods) == 0 {
		//lint:ignore ST1005 German error message requires capitalization
		return fmt.Errorf("Keine SSH-Authentifizierungsmethode konfiguriert (ssh_key_file oder ssh_password erforderlich)")
	}

	// Setup SSH config
	sshConfig := &ssh.ClientConfig{
		User:            config.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	// Default port
	port := config.SSHPort
	if port == "" {
		port = "22"
	}

	// Connect to SSH server
	addr := fmt.Sprintf("%s:%s", config.SSHHost, port)
	logAndPrint(fmt.Sprintf("Verbinde zu SSH-Server: %s", addr))

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("SSH-Verbindung fehlgeschlagen: %v", err)
	}
	defer client.Close()
	logAndPrint("SSH-Verbindung erfolgreich hergestellt")

	// Parse download URL to get remote path
	remoteLocalPath, err := parseRemotePath(updateInfo.DownloadURL, config.SSHDirBase)
	if err != nil {
		return err
	}

	logAndPrint(fmt.Sprintf("Remote-Pfad: %s", remoteLocalPath))

	// Create remote directory if it doesn't exist
	err = createRemoteDir(client, remoteLocalPath)
	if err != nil {
		logAndPrint(fmt.Sprintf("Warnung: Konnte Remote-Verzeichnis nicht erstellen: %v", err))
	}
	// Upload ZIP file using SFTP
	err = uploadFileViaSFTP(client, zipPath, filepath.Join(remoteLocalPath, filepath.Base(zipPath)))
	if err != nil {
		return fmt.Errorf("ZIP-Upload fehlgeschlagen: %v", err)
	}

	// Upload update_info.json using SFTP
	err = uploadFileViaSFTP(client, updateInfoPath, filepath.Join(remoteLocalPath, "update_info.json"))
	if err != nil {
		return fmt.Errorf("update_info.json Upload fehlgeschlagen: %v", err)
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
		//lint:ignore ST1005 German error message requires capitalization
		return "", fmt.Errorf("%s endet in einem Verzeichnis!", downloadURL)
	}
	pos := strings.LastIndex(path, "/")
	if pos <= 0 {
		//lint:ignore ST1005 German error message requires capitalization
		return "", fmt.Errorf("%s enthält keinen Dateinamen!", downloadURL)
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

	logAndPrint(fmt.Sprintf("Remote-Verzeichnis erstellt: %s", remotePath))
	return nil
}

func uploadFileViaSFTP(client *ssh.Client, localPath, remotePath string) error {
	logAndPrint(fmt.Sprintf("Lade Datei hoch: %s -> %s", localPath, remotePath))

	// Create SFTP session
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	// Create remote file using cat command
	remoteDir := filepath.Dir(remotePath)
	remoteDir = filepath.ToSlash(remoteDir)
	remotePath = filepath.ToSlash(remotePath)
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
		stdin.Close()
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

	logAndPrint(fmt.Sprintf("Datei erfolgreich hochgeladen: %s", filepath.Base(localPath)))
	return nil
}
