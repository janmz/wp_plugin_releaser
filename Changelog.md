# Changelog

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
