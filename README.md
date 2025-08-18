# wp_plugin_release

![Go Version](https://img.shields.io/github/go-mod/go-version/janmz/wp_plugin_release)
![Release](https://img.shields.io/github/v/release/janmz/wp_plugin_release)
![License: MIT (modified)](https://img.shields.io/badge/License-MIT--Modified-blue.svg)
[![Support: CFI-Kinderhilfe](https://img.shields.io/badge/Support-CFI--Kinderhilfe-0077B6?logo=heart)](https://cfi-kinderhilfe.de/jetzt-spenden?q=VAYAWPR)
[![Build Status](https://github.com/janmz/wp_plugin_release/workflows/Build%20and%20Release/badge.svg)](https://github.com/janmz/wp_plugin_release/actions)

*[🇩🇪 Deutsche Version](README.de.md) | 🇺🇸 English Version*

A lightweight Go tool for **automated WordPress plugin releases** with full internationalization support, including:

- Automatic version number updates in main PHP file
- Update info management (`update_info.json`)
- ZIP file creation with configurable exclusion patterns
- Optional automatic SSH upload to update server
- **Multi-language support (German/English) with automatic language detection**

## 🌍 Internationalization

This tool supports multiple languages:

- **English** (default)
- **German** (Deutsch)
- **Automatic language detection** based on system locale
- **Extensible** - add more languages by creating JSON files in `locales/`

### Language Override

```bash
# Force German output
LANG=de_DE.UTF-8 wp_plugin_release /path/to/plugin

# Force English output  
LANG=en_US.UTF-8 wp_plugin_release /path/to/plugin
```

## ✨ Features

- **Automatic version detection** (from plugin comment or class variable)
- **Update info management** (`update_info.json`)
- **ZIP creation** with skip patterns
- **SSH upload** (key or password based)
- **Comprehensive logging** with all steps
- **Hardware-bound encryption** for secure password storage
- **Multi-language support** with automatic detection
- **Plugin-update-checker** integration for [YahnisElsts](https://github.com/YahnisElsts/plugin-update-checker)

## 📦 Installation

### Binary Download
Download the latest release for your platform: [Releases](https://github.com/janmz/wp_plugin_release/releases)

### Go Install
```bash
go install github.com/janmz/wp_plugin_release@latest
```

### Docker
```bash
# Pull from GitHub Container Registry
docker pull ghcr.io/janmz/wp_plugin_release:latest

# Or build locally
docker build -t wp_plugin_release .
```

### Build from Source
```bash
git clone https://github.com/janmz/wp_plugin_release.git
cd wp_plugin_release
make build
```

## 🚀 Usage

### Basic Usage
```bash
wp_plugin_release /path/to/plugin
```

- If no path is specified, the current directory is used
- Expects an `update.config` file in the working directory

### Docker Usage
```bash
# Mount current directory and run
docker run --rm -v $(pwd):/workspace wp_plugin_release:latest /workspace

# With German localization
docker run --rm -e LANG=de_DE.UTF-8 -v $(pwd):/workspace wp_plugin_release:latest /workspace

# With SSH keys for upload
docker run --rm \
  -v $(pwd):/workspace \
  -v ~/.ssh:/home/wp_release_user/.ssh:ro \
  wp_plugin_release:latest /workspace
```

## ⚙️ Configuration

### `update.config` Example

```json
{
  "main_php_file": "my-plugin.php",
  "skip_pattern": ["*.psd", "*.bak", "node_modules", ".git"],
  "ssh_host": "example.com",
  "ssh_port": "22", 
  "ssh_dir_base": "/var/www/html/updates",
  "ssh_user": "username",
  "ssh_key_file": "/path/to/key.pem",
  "ssh_password": "password in plain text",
  "ssh_secure_password": "will contain encrypted password after first run"
}
```

### Configuration Fields

| Field | Description | Required |
|-------|-------------|----------|
| `main_php_file` | Main PHP file of the plugin | ✅ |
| `skip_pattern` | Files/directories to exclude from ZIP | ❌ |
| `ssh_host` | SSH hostname for upload | ❌ |
| `ssh_port` | SSH port (default: 22) | ❌ |
| `ssh_dir_base` | Base directory on server | ❌ |
| `ssh_user` | SSH username | ❌ |
| `ssh_key_file` | Path to SSH private key | ❌ |
| `ssh_password` | SSH password (encrypted after first use) | ❌ |

## 🔒 Security Features

- **Hardware-bound encryption**: Passwords are encrypted with a key derived from your system's hardware
- **Automatic password encryption**: Plain text passwords are automatically encrypted after first use
- **Secure file handling**: Backup files are created before modifications
- **SSH key authentication**: Supports both key and password authentication

## 🏗️ Development

### Setup Development Environment
```bash
make setup
```

### Build
```bash
# Standard build
make build

# Build for all platforms
make build-all

# Build with Docker
make docker
```

### Testing
```bash
# Run tests
make test

# Test internationalization
make test-i18n

# Validate translations
make i18n-validate
```

### Internationalization Development

#### Extract Translation Keys
```bash
make i18n-extract
```

#### Add New Language
1. Create `locales/[lang_code].json` (e.g., `locales/fr.json`)
2. Copy structure from `locales/en.json`
3. Translate all values
4. Test with `LANG=[lang_code] wp_plugin_release --help`

#### Translation File Structure
```json
{
  "app.name": "WordPress Plugin Release Tool",
  "app.version": "Version %s from %s started",
  "error.no_directory": "Directory %s does not exist",
  "log.processing_php": "Processing PHP file: %s"
}
```

## 📝 Release Workflow

### Automated Release
1. **Tag**: `git tag v1.0.0` → `git push origin v1.0.0`
2. **GitHub Actions** builds binaries for Linux/macOS/Windows and creates release with assets
3. **Docker images** are automatically built and pushed to GitHub Container Registry

### Manual Release
```bash
make release
```

## 📋 Requirements

### Runtime
- No dependencies (static binary)
- Optional: SSH client for uploads

### Development
- Go 1.21+
- Make (for build automation)
- Docker (optional)
- Git

## 🐳 Docker Support

### Supported Platforms
- `linux/amd64`
- `linux/arm64`

### Environment Variables
| Variable | Description | Default |
|----------|-------------|---------|
| `LANG` | Language locale | `en_US.UTF-8` |
| `WP_PLUGIN_RELEASE_LOCALES_PATH` | Path to translation files | `/usr/local/share/wp_plugin_release/locales` |

## 🤝 Contributing

Contributions are welcome! Please check `CONTRIBUTING.md` before creating a pull request.

### Translation Contributions
We especially welcome contributions for additional languages:
1. Fork the repository
2. Add your language file in `locales/[lang_code].json`
3. Test the translation
4. Submit a pull request

## 📄 License

This software is under a modified MIT license (see `LICENSE`).
You may freely use, modify, and distribute the code, **provided** you credit the original author
**Jan Neuhaus** and maintain a link to the original repository: `https://github.com/janmz/wp_plugin_release`.

**No warranty** is provided.

## 💖 Support

If you find this project helpful, please support **CFI-Kinderhilfe**: https://cfi-kinderhilfe.de/jetzt-spenden?q=VAYAWPR
(Donations go to CFI-Kinderhilfe, not the author.)

## 📞 Contact

**Author**: Jan Neuhaus – VAYA Consulting  
**Website**: https://vaya-consulting.de/development?q=GITHUB  
**Repository**: https://github.com/janmz/wp_plugin_release

## 📚 Additional Resources

- [Plugin Update Checker by YahnisElsts](https://github.com/YahnisElsts/plugin-update-checker)
- [WordPress Plugin Development Handbook](https://developer.wordpress.org/plugins/)
- [Semantic Versioning](https://semver.org/)

## 🔄 Changelog

### v1.1.0 (Current)
- ✨ Full internationalization support (German/English)
- ✨ Automatic language detection
- ✨ Docker support with multi-arch builds
- ✨ Enhanced CI/CD pipeline
- ✨ Improved error handling and logging
- 🐛 Various bug fixes and improvements

### v1.0.0
- 🎉 Initial release
- ✅ Basic plugin release functionality
- ✅ SSH upload support
- ✅ Hardware-bound encryption