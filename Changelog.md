# Changelog

## 2026-02-21

- Security hardening for SSH uploads: optional host key verification via
  `ssh_key_file` when the referenced file exists and is in `known_hosts`
  format.

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

- Increased build version to `1.2.10.48`.
