# Changelog

Alle wesentlichen Änderungen an diesem Projekt werden in dieser Datei
dokumentiert.

## [Unreleased]

### Geändert

- **CI:** Test- und Build-Workflow nutzt lokales `replace` für
  `github.com/janmz/sconfig`. Vor `go mod download` wird sconfig in den
  übergeordneten Verzeichnispfad geklont, damit der Replace-Pfad
  `../sconfig` in GitHub Actions existiert.
- Bei privatem sconfig-Repo kann das Secret `REPO_ACCESS_TOKEN` (PAT mit
  repo-Scope) gesetzt werden; andernfalls wird `GITHUB_TOKEN` verwendet
  (nur für das aktuelle Repo).
