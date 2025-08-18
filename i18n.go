package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// I18n manages internationalization
type I18n struct {
	currentLang string
	messages    map[string]map[string]string
	fallback    string
}

var i18n *I18n

// Initialize i18n system
func initI18n() error {
	i18n = &I18n{
		messages: make(map[string]map[string]string),
		fallback: "en",
	}

	// Detect system language
	lang := detectLanguage()

	// Load translations
	if err := i18n.loadTranslations(); err != nil {
		return fmt.Errorf("failed to load translations: %v", err)
	}

	i18n.setLanguage(lang)
	return nil
}

// detectLanguage tries to detect system language
func detectLanguage() string {
	// Try environment variables
	for _, env := range []string{"LANG", "LC_ALL", "LC_MESSAGES"} {
		if lang := os.Getenv(env); lang != "" {
			if strings.HasPrefix(strings.ToLower(lang), "de") {
				return "de"
			}
			return "en"
		}
	}

	// Default to English
	return "en"
}

// loadTranslations loads translation files from embedded data or external files
func (i *I18n) loadTranslations() error {
	// Load English translations (embedded)
	enTranslations := map[string]string{
		"app.name":                      "WordPress Plugin Release Tool",
		"app.version":                   "Version %s from %s started",
		"app.working_directory":         "Working directory: %s",
		"app.release_process_completed": "Release process completed successfully",
		"app.password_message":          "Enter new password here",

		"error.no_directory":             "Directory %s does not exist or is not readable",
		"error.no_config":                "Directory %s does not contain update.config file",
		"error.config_read":              "Error reading configuration: %v",
		"error.update_info_read":         "Error reading update info: %v",
		"error.php_processing":           "Error processing PHP file: %v",
		"error.update_info_processing":   "Error processing update info: %v",
		"error.zip_creation":             "Error creating ZIP file: %v",
		"error.upload":                   "Error during upload: %v",
		"error.log_file":                 "Error opening log file: %v",
		"error.current_directory":        "Error determining current directory: %v",
		"error.no_valid_version":         "No valid version found",
		"error.no_valid_puc":             "No valid Puc integration found, please revise file %s",
		"error.rename_file":              "Old file could not be renamed: %v",
		"error.write_php":                "PHP file could not be written: %v",
		"error.update_info_missing":      "update_info.json is missing",
		"error.update_info_read_file":    "update_info.json could not be read: %v",
		"error.update_info_invalid_json": "update_info.json has invalid JSON format: %v",
		"error.update_info_structure":    "Structure of update_info.json could not be analyzed: %v",
		"error.zip_create":               "ZIP file could not be created: %v",
		"error.walk_files":               "Error walking through files: %v",
		"error.ssh_no_auth":              "No SSH authentication method configured (ssh_key_file or ssh_password required)",
		"error.ssh_connection":           "SSH connection failed: %v",
		"error.zip_upload":               "ZIP upload failed: %v",
		"error.update_info_upload":       "update_info.json upload failed: %v",
		"error.banner_upload":            "Banner upload failed: %v",
		"error.icon_upload":              "Icon upload failed: %v",
		"error.url_ends_directory":       "%s ends in a directory!",
		"error.url_no_filename":          "%s contains no filename!",
		"error.json_prepare":             "Error preparing JSON data from struct: %v",
		"error.json_mix":                 "Error mixing JSON data: %v",
		"error.json_final":               "Error creating final JSON data: %v",
		"error.backup_create":            "Error creating backup for update_info.json: %v",

		"config.hardware_id_failed":  "Config: Hardware ID cannot be determined",
		"config.default_error":       "Error reading default version of config file: %v",
		"config.default_unsupported": "Unsupported type for default value: %v",
		"config.decrypt_failed":      "Failed to decrypt %s password: %v",

		"log.processing_php":              "Processing PHP file: %s",
		"log.version_comment_found":       "Version found in plugin comment: %s",
		"log.version_class_found":         "Version found in class property: %s",
		"log.version_comment_updated":     "Plugin comment version updated to: %s",
		"log.version_class_updated":       "Class property version updated to: %s",
		"log.last_update_updated":         "Last-Update comment updated to: %s",
		"log.last_update_added":           "Last-Update comment added: %s",
		"log.puc_download_url":            "Download URL found in PUC call: %s",
		"log.puc_comment":                 "Comment found in PUC call: %s",
		"log.puc_slug":                    "Slug found in PUC call: %s",
		"log.puc_changed":                 "Puc integration changed to: %s",
		"log.php_updated":                 "PHP file successfully updated",
		"log.reading_update_info":         "Reading update info: %s",
		"log.current_version_update_info": "Current version in update_info.json: %s",
		"log.processing_update_info":      "Processing update info",
		"log.update_info_updated":         "update_info.json updated to version %s",
		"log.update_info_backup":          "Backup of update_info.json created: %s",
		"log.creating_zip":                "Creating ZIP file: %s",
		"log.skip_patterns":               "Skip patterns: %v",
		"log.skip_directory":              "Skipping directory: %s",
		"log.skip_file":                   "Skipping file: %s",
		"log.file_added":                  "File added: %s",
		"log.zip_created":                 "ZIP file successfully created",
		"log.ssh_upload_start":            "Starting SSH upload",
		"log.ssh_key_warning":             "Warning: SSH key could not be read: %v",
		"log.ssh_key_parse_warning":       "Warning: SSH key could not be parsed: %v",
		"log.ssh_key_added":               "SSH key authentication added",
		"log.ssh_password_added":          "SSH password authentication added",
		"log.ssh_connecting":              "Connecting to SSH server: %s",
		"log.ssh_connected":               "SSH connection successfully established",
		"log.remote_path":                 "Remote path: %s",
		"log.remote_dir_warning":          "Warning: Could not create remote directory: %v",
		"log.banner_not_found":            "Warning: Banner file for entry \"%s\" not found: %s",
		"log.banner_no_url":               "Warning: Banner entry \"%s\" is not a URL (%s)",
		"log.icon_not_found":              "Warning: Icon file for entry \"%s\" not found: %s",
		"log.icon_no_url":                 "Warning: Icon entry \"%s\" is not a URL (%s)",
		"log.uploading_file":              "Uploading file: %s -> %s",
		"log.file_uploaded":               "File successfully uploaded: %s",
		"log.remote_dir_created":          "Remote directory created: %s",
		"log.no_ssh_config":               "No SSH configuration found, upload skipped",
		"log.upload_completed":            "Upload completed successfully",
		"log.current_version_detected":    "Current version detected: %s",
		"log.update_info_version_updated": "update_info.json version updated to: %s",
		"log.download_url_set":            "Download URL set to '%s'!",
	}

	// Load German translations (embedded)
	deTranslations := map[string]string{
		"app.name":                      "WordPress Plugin Release Tool",
		"app.version":                   "Version %s vom %s gestartet",
		"app.working_directory":         "Arbeitsverzeichnis: %s",
		"app.release_process_completed": "Release-Prozess erfolgreich abgeschlossen",
		"app.password_message":          "Hier neues Passwort eintragen",

		"error.no_directory":             "Das Verzeichnis %s existiert nicht oder ist nicht lesbar",
		"error.no_config":                "Das Verzeichnis %s enthält keine Datei update.config",
		"error.config_read":              "Fehler beim Lesen der Konfiguration: %v",
		"error.update_info_read":         "Fehler beim Lesen der Update-Info: %v",
		"error.php_processing":           "Fehler beim Verarbeiten der PHP-Datei: %v",
		"error.update_info_processing":   "Fehler beim Anpassen der Update-Info: %v",
		"error.zip_creation":             "Fehler beim Erstellen der ZIP-Datei: %v",
		"error.upload":                   "Fehler beim Upload: %v",
		"error.log_file":                 "Fehler beim Öffnen der Log-Datei: %v",
		"error.current_directory":        "Fehler beim Ermitteln des aktuellen Verzeichnisses: %v",
		"error.no_valid_version":         "Keine gültige Version gefunden",
		"error.no_valid_puc":             "Keine gültige Puc-Integration gefunden, bitte Datei %s überarbeiten",
		"error.rename_file":              "Alte Datei konnte nicht umbenannt werden: %v",
		"error.write_php":                "PHP-Datei konnte nicht geschrieben werden: %v",
		"error.update_info_missing":      "update_info.json ist nicht vorhanden",
		"error.update_info_read_file":    "update_info.json konnte nicht gelesen werden: %v",
		"error.update_info_invalid_json": "update_info.json hat ungültiges JSON-Format: %v",
		"error.update_info_structure":    "Struktur von update_info.json konnte nicht analysiert werden: %v",
		"error.zip_create":               "ZIP-Datei konnte nicht erstellt werden: %v",
		"error.walk_files":               "Fehler beim Durchlaufen der Dateien: %v",
		"error.ssh_no_auth":              "Keine SSH-Authentifizierungsmethode konfiguriert (ssh_key_file oder ssh_password erforderlich)",
		"error.ssh_connection":           "SSH-Verbindung fehlgeschlagen: %v",
		"error.zip_upload":               "ZIP-Upload fehlgeschlagen: %v",
		"error.update_info_upload":       "update_info.json Upload fehlgeschlagen: %v",
		"error.banner_upload":            "Banner-Upload fehlgeschlagen: %v",
		"error.icon_upload":              "Icon-Upload fehlgeschlagen: %v",
		"error.url_ends_directory":       "%s endet in einem Verzeichnis!",
		"error.url_no_filename":          "%s enthält keinen Dateinamen!",
		"error.json_prepare":             "Fehler beim Vorbereiten der JSON-Daten aus dem Struct: %v",
		"error.json_mix":                 "Fehler beim Mischen der JSON-Daten: %v",
		"error.json_final":               "Fehler beim Erstellen der finalen JSON-Daten: %v",
		"error.backup_create":            "Fehler beim Erstellen der Sicherung für update_info.json: %v",

		"config.hardware_id_failed":  "Config: Hardware ID kann nicht bestimmt werden",
		"config.default_error":       "Fehler beim Auslesen der default-Version der config-Datei: %v",
		"config.default_unsupported": "Nicht unterstützter Typ für Standard-Wert: %v",
		"config.decrypt_failed":      "Entschlüsselung des %s Passworts fehlgeschlagen: %v",

		"log.processing_php":              "Verarbeite PHP-Datei: %s",
		"log.version_comment_found":       "Version im Plugin-Kommentar gefunden: %s",
		"log.version_class_found":         "Version in Klassen-Property gefunden: %s",
		"log.version_comment_updated":     "Plugin-Kommentar Version aktualisiert auf: %s",
		"log.version_class_updated":       "Klassen-Property Version aktualisiert auf: %s",
		"log.last_update_updated":         "Last-Update Kommentar aktualisiert auf: %s",
		"log.last_update_added":           "Last-Update Kommentar hinzugefügt: %s",
		"log.puc_download_url":            "Im PUC-Aufruf DownloadURL gefunden: %s",
		"log.puc_comment":                 "Im PUC-Aufruf Kommentar gefunden: %s",
		"log.puc_slug":                    "Im PUC Aufruf Slug gefunden: %s",
		"log.puc_changed":                 "Puc-Integration geändert zu: %s",
		"log.php_updated":                 "PHP-Datei erfolgreich aktualisiert",
		"log.reading_update_info":         "Einlesen Update-Info: %s",
		"log.current_version_update_info": "Aktuelle Version in update_info.json: %s",
		"log.processing_update_info":      "Verarbeite Update-Infos",
		"log.update_info_updated":         "update_info.json aktualisiert auf Version %s",
		"log.update_info_backup":          "Sicherung von update_info.json erstellt: %s",
		"log.creating_zip":                "Erstelle ZIP-Datei: %s",
		"log.skip_patterns":               "Skip-Patterns: %v",
		"log.skip_directory":              "Überspringe Verzeichnis: %s",
		"log.skip_file":                   "Überspringe Datei: %s",
		"log.file_added":                  "Datei hinzugefügt: %s",
		"log.zip_created":                 "ZIP-Datei erfolgreich erstellt",
		"log.ssh_upload_start":            "Beginne SSH-Upload",
		"log.ssh_key_warning":             "Warnung: SSH-Schlüssel konnte nicht gelesen werden: %v",
		"log.ssh_key_parse_warning":       "Warnung: SSH-Schlüssel konnte nicht geparst werden: %v",
		"log.ssh_key_added":               "SSH-Schlüssel-Authentifizierung hinzugefügt",
		"log.ssh_password_added":          "SSH-Passwort-Authentifizierung hinzugefügt",
		"log.ssh_connecting":              "Verbinde zu SSH-Server: %s",
		"log.ssh_connected":               "SSH-Verbindung erfolgreich hergestellt",
		"log.remote_path":                 "Remote-Pfad: %s",
		"log.remote_dir_warning":          "Warnung: Konnte Remote-Verzeichnis nicht erstellen: %v",
		"log.banner_not_found":            "Warnung: Banner-Datei für Eintrag \"%s\" nicht gefunden: %s",
		"log.banner_no_url":               "Warnung: Banner-Eintrag \"%s\" ist keine URL (%s)",
		"log.icon_not_found":              "Warnung: Icon-Datei für Eintrag \"%s\" nicht gefunden: %s",
		"log.icon_no_url":                 "Warnung: Icon-Eintrag \"%s\" ist keine URL (%s)",
		"log.uploading_file":              "Lade Datei hoch: %s -> %s",
		"log.file_uploaded":               "Datei erfolgreich hochgeladen: %s",
		"log.remote_dir_created":          "Remote-Verzeichnis erstellt: %s",
		"log.no_ssh_config":               "Keine SSH-Konfiguration gefunden, Upload übersprungen",
		"log.upload_completed":            "Upload erfolgreich abgeschlossen",
		"log.current_version_detected":    "Aktuelle Version ermittelt: %s",
		"log.update_info_version_updated": "update_info.json Version aktualisiert auf: %s",
		"log.download_url_set":            "Download URL auf '%s' gesetzt!",
	}

	i.messages["en"] = enTranslations
	i.messages["de"] = deTranslations

	// Try to load external translation files
	i.loadExternalTranslations()

	return nil
}

// loadExternalTranslations tries to load translation files from locales directory
func (i *I18n) loadExternalTranslations() {
	localesDir := "locales"
	if _, err := os.Stat(localesDir); os.IsNotExist(err) {
		return
	}

	files, err := filepath.Glob(filepath.Join(localesDir, "*.json"))
	if err != nil {
		return
	}

	for _, file := range files {
		lang := strings.TrimSuffix(filepath.Base(file), ".json")

		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var translations map[string]string
		if err := json.Unmarshal(data, &translations); err != nil {
			continue
		}

		// Merge with existing translations
		if i.messages[lang] == nil {
			i.messages[lang] = make(map[string]string)
		}

		for key, value := range translations {
			i.messages[lang][key] = value
		}
	}
}

// setLanguage sets the current language
func (i *I18n) setLanguage(lang string) {
	if _, exists := i.messages[lang]; exists {
		i.currentLang = lang
	} else {
		i.currentLang = i.fallback
	}
}

// translate translates a key to the current language
func (i *I18n) translate(key string, args ...interface{}) string {
	// Try current language
	if msg, exists := i.messages[i.currentLang][key]; exists {
		if len(args) > 0 {
			return fmt.Sprintf(msg, args...)
		}
		return msg
	}

	// Try fallback language
	if msg, exists := i.messages[i.fallback][key]; exists {
		if len(args) > 0 {
			return fmt.Sprintf(msg, args...)
		}
		return msg
	}

	// Return key if no translation found
	if len(args) > 0 {
		return fmt.Sprintf(key+": %v", args)
	}
	return key
}

// Helper functions for easy access
func t(key string, args ...interface{}) string {
	if i18n == nil {
		if len(args) > 0 {
			return fmt.Sprintf(key+": %v", args)
		}
		return key
	}
	return i18n.translate(key, args...)
}

// getCurrentLanguage returns the current language code
func getCurrentLanguage() string {
	if i18n == nil {
		return "en"
	}
	return i18n.currentLang
}

// setLanguage sets the language (public interface)
func setLanguage(lang string) {
	if i18n != nil {
		i18n.setLanguage(lang)
	}
}
