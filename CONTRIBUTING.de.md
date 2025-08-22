# Mitwirken bei wp_plugin_release

*[🇺🇸 Englische Version](CONTRIBUTING.md) | 🇩🇪 Deutsche Version*

Vielen Dank für dein Interesse, zu wp_plugin_release beizutragen! Dieses Dokument enthält Richtlinien für die Mitarbeit an diesem internationalisierten Go-Projekt.

## Beiträge zu Übersetzungen

Wir freuen uns besonders über Beiträge zu Übersetzungen! Derzeit unterstützte Sprachen:
- Englisch (en)
- Deutsch (de)

### Eine neue Sprache hinzufügen

1.  **Forke das Repository**
2.  **Erstelle eine neue Übersetzungsdatei**:
    ```bash
    cp locales/en.json locales/[dein_sprachcode].json
    ```
3.  **Übersetze alle Einträge** in deiner Sprachdatei
4.  **Teste die Übersetzung**:
    ```bash
    LANG=[dein_sprachcode] ./bin/wp_plugin_release --help
    ```
5.  **Reiche einen Pull-Request ein** mit:
    - Deiner Übersetzungsdatei
    - Aktualisierter README mit deiner aufgelisteten Sprache
    - Kurzer Beschreibung der Sprache/des Gebietsschemas

### Übersetzungsrichtlinien

-   **Verwende die angemessene Anredeform** für deine Sprache/Kultur
-   **Halte technische Begriffe konsistent** (z. B. "ZIP-Datei", "SSH")
-   **Behalte Formatierungszeichenketten** wie `%s`, `%v`, `%d` bei
-   **Teste mit tatsächlichen Fehlerszenarien**, um sicherzustellen, dass die Übersetzungen funktionieren
-   **Berücksichtige den Kontext** – einige Begriffe benötigen möglicherweise in verschiedenen Kontexten unterschiedliche Übersetzungen

### Struktur der Übersetzungsdatei

```json
{
  "app.name": "Deine Übersetzung hier",
  "app.version": "Version %s von %s gestartet",
  "error.no_directory": "Verzeichnis %s existiert nicht",
  "log.processing_php": "Verarbeite PHP-Datei: %s"
}
```

**Wichtig**:
-   Behalte Formatbezeichner (`%s`, `%v`) an derselben Position
-   Übersetze nicht die Schlüssel (linke Seite), sondern nur die Werte (rechte Seite)
-   Stelle die Gültigkeit der JSON-Syntax sicher

## Beiträge zum Code

### Entwicklungsumgebung einrichten

1.  **Forken und klonen**:
    ```bash
    git clone https://github.com/dein-benutzername/wp_plugin_release.git
    cd wp_plugin_release
    ```

### Änderungen vornehmen

1.  **Erstelle einen Feature-Branch**:
    ```bash
    git checkout -b feature/dein-feature-name
    ```

2.  **Nimm deine Änderungen vor** und befolge dabei unsere Programmierstandards:
    -   Verwende die `t()`-Funktion für alle für den Benutzer sichtbaren Zeichenketten
    -   Füge entsprechende Übersetzungsschlüssel sowohl zu `locales/en.json` als auch zu `locales/de.json` hinzu
    -   Befolge die Go-Konventionen und den bestehenden Programmierstil
    -   Füge Tests für neue Funktionalitäten hinzu

3.  **Teste deine Änderungen**:
    ```bash
    go test -v
    ```

4.  **Linter ausführen**:
    ```bash
    go vet -v
    ```

### Programmierstandards

#### Internationalisierung

-   **Alle für den Benutzer sichtbaren Zeichenketten** müssen die `t()`-Funktion verwenden:
    ```go
    // Gut
    logAndPrint(t("log.processing_php", phpFilePath))
    
    // Schlecht
    logAndPrint("Verarbeite PHP-Datei: " + phpFilePath)
    ```

-   **Füge Übersetzungsschlüssel** sowohl zu `locales/en.json` als auch zu `locales/de.json` hinzu

-   **Verwende beschreibende Schlüsselnamen** mit Kategorien:
    ```
    app.* - Anwendungsnachrichten
    error.* - Fehlermeldungen
    log.* - Protokollnachrichten
    config.* - Konfigurationsnachrichten
    ```

#### Go-Programmierstil

-   Befolge die Standard-Go-Formatierung (`go fmt`)
-   Verwende beschreibende Variablennamen
-   Füge Kommentare für exportierte Funktionen hinzu
-   Behandle Fehler angemessen
-   Verwende nach Möglichkeit strukturiertes Logging

### Testen

#### Unit-Tests

```bash
make test
```

#### Integrationstests

```bash
# Test mit einem tatsächlichen Plugin-Verzeichnis
./bin/wp_plugin_release /pfad/zum/test/plugin
```

#### Übersetzungstests

```bash
make test-i18n
```

### Änderungen einreichen

1.  **Commite deine Änderungen**:
    ```bash
    git add .
    git commit -m "feat: füge Unterstützung für französische Übersetzung hinzu"
    ```

2.  **Pushe zu deinem Fork**:
    ```bash
    git push origin feature/dein-feature-name
    ```

3.  **Erstelle einen Pull-Request** mit:
    -   Klarer Beschreibung der Änderungen
    -   Screenshots bei UI-Änderungen
    -   Ergebnissen der Übersetzungstests, falls zutreffend
    -   Verweis auf alle zugehörigen Issues

## Richtlinien für Commit-Nachrichten

Wir folgen den [Conventional Commits](https://www.conventionalcommits.org/):

```
<typ>(<bereich>): <beschreibung>

[optionaler Rumpf]

[optionale(r) Fußzeile(n)]
```

### Typen

-   `feat`: Neues Feature
-   `fix`: Fehlerbehebung
-   `docs`: Änderungen an der Dokumentation
-   `style`: Änderungen am Codestil (Formatierung etc.)
-   `refactor`: Code-Refactoring
-   `test`: Hinzufügen oder Aktualisieren von Tests
-   `chore`: Wartungsaufgaben
-   `i18n`: Änderungen an der Internationalisierung

### Beispiele

```
feat(i18n): füge Unterstützung für französische Übersetzung hinzu
fix(config): behandle fehlende Konfigurationsdatei ordnungsgemäß
docs: aktualisiere Installationsanweisungen
i18n(de): verbessere deutsche Fehlermeldungen
```

## Fehlerberichte

Wenn du Fehler meldest, füge bitte Folgendes hinzu:

1.  **Informationen zur Umgebung**:
    -   Betriebssystem
    -   Go-Version
    -   Sprach-/Gebietsschemaeinstellungen

2.  **Schritte zur Reproduktion**

3.  **Erwartetes vs. tatsächliches Verhalten**

4.  **Fehlermeldungen** (wenn möglich in der Originalsprache)

5.  **Konfigurationsdatei** (bereinigt, entferne sensible Daten)

## Feature-Anfragen

Für Feature-Anfragen:

1.  **Überprüfe zuerst bestehende Issues**
2.  **Beschreibe den Anwendungsfall** deutlich
3.  **Berücksichtige die Auswirkungen auf die Internationalisierung**
4.  **Gib gegebenenfalls Beispiele** an

## Entwicklungsworkflow

### Branch-Strategie

-   `main` – stabile Veröffentlichungen
-   `develop` – Entwicklungsbranch
-   `feature/*` – Feature-Branches
-   `fix/*` – Branches für Fehlerbehebungen
-   `i18n/*` – Übersetzungs-Branches

### Veröffentlichungsprozess

1.  Die Entwicklung findet auf `develop` statt
2.  Features werden über Pull-Requests gemerged
3.  Release-Kandidaten werden als `v1.0.0-rc1` getaggt
4.  Finale Veröffentlichungen werden als `v1.0.0` getaggt
5.  Automatisierte Builds erstellen Binärdateien für alle Plattformen

## Checkliste für Mitwirkende

### Für Code-Änderungen
- [ ] Der Code folgt den Projektkonventionen
- [ ] Alle für den Benutzer sichtbaren Zeichenketten verwenden die `t()`-Funktion
- [ ] Übersetzungsschlüssel wurden zu allen Sprachdateien hinzugefügt
- [ ] Tests sind erfolgreich (`make test`)
- [ ] Linter sind erfolgreich (`make lint`)
- [ ] Die Übersetzungvalidierung ist erfolgreich (`make i18n-validate`)
- [ ] Die Dokumentation wurde bei Bedarf aktualisiert

### Für Übersetzungsänderungen
- [ ] Die Übersetzungsdatei ist valides JSON
- [ ] Alle Schlüssel aus der englischen Version sind übersetzt
- [ ] Formatbezeichner wurden korrekt beibehalten
- [ ] Getestet mit `LANG=<code> wp_plugin_release --help`
- [ ] Kulturelle Angemessenheit wurde berücksichtigt

## Community

-   **Sei respektvoll** und jedem gegenüber wertschätzend
-   **Hilf anderen**, zu lernen und beizutragen
-   **Gib konstruktives Feedback**
-   **Feiere die Vielfalt** in Sprachen und Kulturen

## Hilfe erhalten

-   **Issues**: GitHub Issues für Fehler und Feature-Anfragen
-   **Discussions**: GitHub Discussions für Fragen
-   **E-Mail**: security@vaya-consulting.de für Sicherheitsprobleme

## Lizenz

Durch deine Mitarbeit stimmst du zu, dass deine Beiträge unter derselben modifizierten MIT-Lizenz wie das Projekt lizenziert werden.

---

Vielen Dank, dass du zu wp_plugin_release beiträgst!
