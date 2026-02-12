# WP_Plugin_Releaser

[![Go Version](https://img.shields.io/github/go-mod/go-version/janmz/wp_plugin_releaser)](https://golang.org)
[![Release](https://img.shields.io/github/v/release/janmz/wp_plugin_releaser)](https://github.com/janmz/wp_plugin_releaser/releases)
[![Lizenz: MIT (modifiziert)](https://img.shields.io/badge/Lizenz-MIT--Modified-blue.svg)](LICENSE)
[![Unterstützung: CFI-Kinderhilfe](https://img.shields.io/badge/Unterstützung-CFI--Kinderhilfe-0077B6?logo=heart)](https://cfi-kinderhilfe.de/jetzt-spenden?q=VAYAWPR)
[![Build Status](https://github.com/janmz/wp_plugin_releaser/workflows/Build%20and%20Release/badge.svg)](https://github.com/janmz/wp_plugin_releaser/actions)

*🌍 🇩🇪 Deutsche Version | 🇺🇸 [English Version](README.md)*

**wp_plugin_releaser** ist ein schlankes Go-Tool für **automatisierte
WordPress-Plugin-Releases** mit vollständiger Internationalisierung, u. a.:

- Aktualisierung der Versionsnummer in der Haupt-PHP-Datei
- Update-Info-Management (`update_info.json`)
- ZIP-Erstellung mit konfigurierbaren Ausschlussmustern
- Optionaler automatischer SSH-Upload auf den Update-Server
- **Mehrsprachigkeit (Deutsch/Englisch) mit automatischer Spracherkennung**

## Internationalisierung

Das Tool unterstützt mehrere Sprachen:

- **Englisch** (Standard)
- **Deutsch**
- **Automatische Spracherkennung** anhand der Systemsprache
- **Erweiterbar** – weitere Sprachen über JSON-Dateien in `locales/`

### Sprache erzwingen

```bash
# Deutsche Ausgabe
LANG=de_DE.UTF-8 wp_plugin_release /pfad/zum/plugin

# Englische Ausgabe
LANG=en_US.UTF-8 wp_plugin_release /pfad/zum/plugin
```

## Features

- **Automatische Versionserkennung** (aus Plugin-Kommentar oder Klassenvariable)
- **Update-Info-Management** (`update_info.json`)
- **ZIP-Erstellung** mit Skip-Patterns
- **SSH-Upload** (per Key oder Passwort)
- **Ausführliche Protokollierung** aller Schritte
- **Hardwaregebundene Verschlüsselung** für sichere Passwortspeicherung
- **Mehrsprachigkeit** mit automatischer Erkennung
- **Plugin-update-checker**-Integration von
  [YahnisElsts](https://github.com/YahnisElsts/plugin-update-checker)

## Installation

### Binary herunterladen

Neueste Version für deine Plattform: [Releases](https://github.com/janmz/wp_plugin_release/releases)

### Go Install

```bash
go install github.com/janmz/wp_plugin_release@latest
```

### Aus Quellcode bauen

```bash
git clone https://github.com/janmz/wp_plugin_release.git
cd wp_plugin_release
make build
```

## Verwendung

### Grundlegende Nutzung

```bash
wp_plugin_release /pfad/zum/plugin
```

- Ohne Pfad wird das aktuelle Verzeichnis verwendet.
- Im Arbeitsverzeichnis wird eine `update.config` erwartet.

## Konfiguration

### Beispiel `update.config`

```json
{
  "main_php_file": "mein-plugin.php",
  "skip_pattern": ["*.psd", "*.bak", "node_modules", ".git"],
  "ssh_host": "example.com",
  "ssh_port": "22",
  "ssh_dir_base": "/var/www/html/updates",
  "ssh_user": "username",
  "ssh_key_file": "/pfad/zu/key.pem",
  "ssh_password": "Passwort im Klartext",
  "ssh_secure_password": "wird nach erstem Lauf verschlüsselt übernommen"
}
```

### Konfigurationsfelder

| Feld | Beschreibung | Pflicht |
| ---- | ------------- | ------- |
| `main_php_file` | Haupt-PHP-Datei des Plugins | ✅ |
| `skip_pattern` | Dateien/Verzeichnisse, die nicht ins ZIP sollen | ❌ |
| `ssh_host` | SSH-Host für Upload | ✅ |
| `ssh_port` | SSH-Port (Standard: 22) | ✅ |
| `ssh_dir_base` | Basisverzeichnis auf dem Server | ✅ |
| `ssh_user` | SSH-Benutzername | ✅ |
| `ssh_key_file` | Pfad zum SSH-Private-Key | ❌ |
| `ssh_password` | SSH-Passwort (nach erstem Einsatz verschlüsselt) | ✅ |

## Sicherheitsfunktionen

- **Hardwaregebundene Verschlüsselung**: Passwörter werden mit einem vom
  System abgeleiteten Schlüssel verschlüsselt.
- **Automatische Passwortverschlüsselung**: Klartext-Passwörter werden nach
  der ersten Nutzung automatisch verschlüsselt.
- **Sichere Dateiverarbeitung**: Vor Änderungen werden Backups erstellt.
- **SSH-Key-Authentifizierung**: Unterstützung von Key- und Passwort-Auth.

## Entwicklung

### Entwicklungsumgebung einrichten

```bash
make setup
```

### Build

```bash
# Standard-Build
make build

# Build für alle Plattformen
make build-all
```

### Tests

```bash
# Tests ausführen
make test

# Internationalisierung testen
make test-i18n

# Übersetzungen prüfen
make i18n-validate
```

### Internationalisierung (Entwicklung)

#### Übersetzungsschlüssel extrahieren

```bash
make i18n-extract
```

#### Neue Sprache hinzufügen

1. Datei `locales/[sprachcode].json` anlegen (z. B. `locales/fr.json`)
2. Struktur von `locales/en.json` übernehmen
3. Alle Werte übersetzen
4. Test mit `LANG=[sprachcode] wp_plugin_release --help`

#### Struktur der Übersetzungsdateien

```json
{
  "app.name": "WordPress Plugin Release Tool",
  "app.version": "Version %s vom %s gestartet",
  "error.no_directory": "Verzeichnis %s existiert nicht",
  "log.processing_php": "Verarbeite PHP-Datei: %s"
}
```

## Release-Workflow

### Automatisches Release

1. **Tag setzen**: `git tag v1.0.0` → `git push origin v1.0.0`
2. **GitHub Actions** baut Binaries für Linux/macOS/Windows und erstellt
   das Release inkl. Assets.

### Manuelles Release

```bash
make release
```

## Anforderungen

### Laufzeit

- Keine Abhängigkeiten (statisches Binary)
- Optional: SSH-Client für Uploads

### Entwicklungsumgebung

- Go 1.21+
- Make (für Build-Automatisierung)
- Git

## Contributing

Beiträge sind willkommen! Bitte vor einem Pull Request `CONTRIBUTING.md`
lesen.

### Übersetzungsbeiträge

Besonders willkommen sind Beiträge für weitere Sprachen:

1. Repository forken
2. Sprachdatei in `locales/[sprachcode].json` anlegen
3. Übersetzung testen
4. Pull Request einreichen

## Lizenz

Diese Software steht unter einer modifizierten MIT-Lizenz (siehe `LICENSE`).
Du darfst den Code frei verwenden, anpassen und weitergeben, **solange** du
den ursprünglichen Autor **Jan Neuhaus** nennst und einen Link auf das
Original-Repository beibehältst: `https://github.com/janmz/wp_plugin_release`.

**Es wird keine Gewährleistung übernommen.**

## Unterstützung

Wenn dir das Projekt nützt, unterstütze bitte die **CFI-Kinderhilfe**:
[Spendenseite](https://cfi-kinderhilfe.de/jetzt-spenden?q=VAYAWPR)  
(Spenden gehen an die CFI-Kinderhilfe, nicht an den Autor.)

## Kontakt

**Autor**: Jan Neuhaus – [VAYA Consulting](https://vaya-consulting.de/development?q=GITHUB)
**Repository**: [https://github.com/janmz/wp_plugin_release]

## Weitere Ressourcen

- [Plugin Update Checker von YahnisElsts](https://github.com/YahnisElsts/plugin-update-checker)
- [WordPress Plugin Development Handbook](https://developer.wordpress.org/plugins/)

## Changelog

### v1.1.0 (aktuell)

- Vollständige Internationalisierung (Deutsch/Englisch)
- Automatische Spracherkennung
- Erweiterte CI/CD-Pipeline
- Verbesserte Fehlerbehandlung und Protokollierung
- Diverse Bugfixes und Verbesserungen

### v1.0.0

- Erste Veröffentlichung
- Grundlegende Plugin-Release-Funktionen
- SSH-Upload-Unterstützung
- Hardwaregebundene Verschlüsselung
