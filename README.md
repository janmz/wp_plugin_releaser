# wp_plugin_release

![Go Version](https://img.shields.io/github/go-mod/go-version/USERNAME/wp_plugin_release)
![Release](https://img.shields.io/github/v/release/USERNAME/wp_plugin_release)
![License: MIT (modified)](https://img.shields.io/badge/License-MIT--Modified-blue.svg)
[![Support: CFI-Kinderhilfe](https://img.shields.io/badge/Support-CFI--Kinderhilfe-0077B6?logo=heart)](https://cfi-kinderhilfe.de)

Ein leichtgewichtiges Go-Tool, um **neue Releases für WordPress-Plugins** automatisiert bereitzustellen – inklusive:

- Aktualisierung der Versionsnummer in der Haupt-PHP-Datei
- Anpassung der `update_info.json`
- Erstellen einer ZIP-Datei mit konfigurierbaren Ausschlussmustern
- Optionaler automatischer Upload via SSH auf den Update-Server

## Features

- **Automatische Versionserkennung** (aus Plugin-Kommentar oder Klassenvariable)
- **Update-Info-Management** (`update_info.json`)
- **ZIP-Erstellung** mit Skip-Patterns
- **SSH-Upload** (Key- oder Passwort-basiert)
- **Logdatei** mit allen Schritten

## Installation

```bash
go install github.com/USERNAME/wp_plugin_release@latest
```

Oder Release-Binaries herunterladen: [Releases](https://github.com/USERNAME/wp_plugin_release/releases)

## Verwendung

```bash
wp_plugin_release /pfad/zum/plugin
```

- Falls kein Pfad angegeben, wird das aktuelle Verzeichnis verwendet.
- Erwartet eine `update.config` im Arbeitsverzeichnis.

## `update.config` Beispiel

```json
{
  "main_php_file": "mein-plugin.php",
  "skip_pattern": ["*.psd", "*.bak"],
  "ssh_host": "example.com",
  "ssh_port": "22",
  "ssh_dir_base": "/var/www/html/updates",
  "ssh_user": "username",
  "ssh_key_file": "/pfad/zu/key.pem",
  "ssh_password": "",
  "ssh_secure_password": ""
}
```

## Release-Workflow (kurz)

- Taggen: `git tag v1.0.0` → `git push origin v1.0.0`
- GitHub Actions baut Binaries für Linux/Mac/Windows und erstellt ein Release mit Assets.

## Lizenz

Diese Software steht unter einer modifizierten MIT-Lizenz (siehe `LICENSE`).
Du darfst den Code frei verwenden, anpassen und weitergeben, **solange** du den ursprünglichen Autor
**Jan Neuhaus** nennst und einen Link auf das Original-Repository beibehältst: `https://github.com/USERNAME/wp_plugin_release`.

Es wird **keinerlei Gewährleistung** übernommen.

## Spenden

Wenn Ihnen das Projekt gefällt, unterstützten Sie bitte die **CFI-Kinderhilfe**: https://cfi-kinderhilfe.de
(Spenden gehen an die CFI-Kinderhilfe, nicht an den Autor.)

## Contributing

Beiträge sind willkommen! Bitte schaue dir `CONTRIBUTING.md` an, bevor du einen Pull Request erstellst.

## Kontakt

Author: Jan Neuhaus — VAYA Consulting
