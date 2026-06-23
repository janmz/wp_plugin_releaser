# Changelog

## 2026-06-23 09:52:10

- Refactor: SSH upload now uses `github.com/janmz/ssh-commands` instead of
  local `ssh.go` SFTP code.
- Change: SSH host key verification is always required; use `ssh_known_hosts`
  (or `~/.ssh/known_hosts`) or start with `-fetch-hostkey` to fetch and store
  the server key. `-trustserver` remains as an alias for `-fetch-hostkey`.

## 2026-06-22 14:40:32

- Fix: GitHub Actions split into `ci.yml` (branches/PR) and `release.yml` (tags
  only) to avoid duplicate workflow runs when pushing to `main` and a `v*` tag.
  Tagged commits on `main` skip the CI workflow; the release workflow handles
  them.

## 2026-06-22 12:35:26

- Change: Local CI checks and pre-commit hook use `go run ./scripts/cicheck`
  (pure Go, no bash). On Windows: `scripts/install-hooks.ps1` or `make
  install-hooks` builds `.githooks/pre-commit.exe`; on Unix a shell hook runs
  `go run ./scripts/cicheck`.

## 2026-06-22 12:28:09

- Fix: CI workflow aligned with Go 1.25 (`go.mod`); golangci-lint now built via
  `goinstall` to avoid version mismatch.
- Fix: Go module cache path and bash `if`-syntax in the build workflow.
- Feature: Pre-commit hook (`.githooks/pre-commit`) and `scripts/ci-check.sh`
  mirror CI tests, i18n validation, and lint locally.

## 2026-04-28 10:12:54

- Fix: When appending a changelog note for an existing version, preserve the
  existing section and prepend the new note as a bullet at the top (matching
  `## [x.y.z]` and `## x.y.z` headers).

## 2026-04-28 10:23:53

- Change: SVG→PNG generation now creates a single `*-h1024.png` (height 1024px,
  preserving aspect ratio) for files that are neither `banner*` nor
  `logo*`/`icon*`.

## 2026-04-28 10:42:57

- Refactor: Moved the application entrypoint (`main.go`) back into
  `wp_plugin_release.go` and removed `main.go`.

## 2026-04-15 21:01:02

- Fix: `update_info.json` changelog now uses the latest 5 entries from
  `Changelog.md`, formatted as a single-line HTML `<dl>` without line breaks.

## 2026-04-15 12:09:07

- Fix: If `ssh_known_hosts` (or default `~/.ssh/known_hosts`) is missing, the
  SSH host key is fetched from the server on first connect and written to the
  known_hosts file, enabling host key verification afterwards. This behavior
  now only happens when started with `-trustserver`.

- Fix: When started with `-trustserver` and a connection fails with
  `knownhosts: key mismatch`, the SSH host key is fetched from the server,
  appended to the used `known_hosts` file and the connection is retried once.

- Fix: When started with `-c <msg>` or `-commit <msg>`, the given message is
  used for the git commit message instead of interactive input.

## 2026-04-15 11:47:24

- Fix: SVG→PNG conversion now also runs when expected PNGs exist but are older
  than the source SVG (stale outputs). Logging now distinguishes missing vs.
  stale PNG targets.

## 2026-04-13 11:03:55

- Fix: SVG→PNG conversion now runs when expected PNGs are missing (even if no
  SVG file is detected as changed), with detailed logging of conversion
  candidates and missing PNG targets.

## 2026-02-21

- Security fix: SSH host key verification now uses `ssh_known_hosts` (new
  config field) or `~/.ssh/known_hosts` by default. Previously, `ssh_key_file`
  (private key path) was incorrectly used as known_hosts, so host verification
  was effectively never enabled.

- Replaced remote shell command usage with native SFTP operations for directory
  creation, file upload, and remote modification time checks.

- Added path traversal protection for `main_php_file`.

- Added ZIP slug validation to prevent path injection / zip-slip style archive
  paths.

- Reduced sensitive URL exposure in logs by removing query and fragment parts.

- Added `update.config` to `.gitignore` to avoid accidentally committing local
  secrets.

- Updated `README.md` and `README.de.md` with host key setup instructions and
  config examples.

- Added `ssh_known_hosts` config field; documented in README.

- Increased build version to `1.2.11.50`.
