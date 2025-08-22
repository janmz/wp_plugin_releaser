# Contributing to wp_plugin_release

*[🇩🇪 Deutsche Version](CONTRIBUTING.de.md) | 🇺🇸 English Version*

Thank you for your interest in contributing to wp_plugin_release! This document provides guidelines for contributing to this internationalized Go project.

## Translation Contributions

We especially welcome translation contributions! Currently supported languages:

- English (en)
- German (de)

### Adding a New Language

1. **Fork the repository**
2. **Create a new translation file**:

   ```bash
   cp locales/en.json locales/[your_lang_code].json
   ```

3. **Translate all entries** in your language file
4. **Test the translation**:

   ```bash
   LANG=[your_lang_code] ./bin/wp_plugin_release --help
   ```

5. **Submit a pull request** with:
   - Your translation file
   - Updated README with your language listed
   - Brief description of the language/locale

### Translation Guidelines

- **Use appropriate formality levels** for your language/culture
- **Keep technical terms consistent** (e.g., "ZIP file", "SSH")
- **Preserve format strings** like `%s`, `%v`, `%d`
- **Test with actual error scenarios** to ensure translations work
- **Consider context** - some terms might need different translations in different contexts

### Translation File Structure

```json
{
  "app.name": "Your translation here",
  "app.version": "Version %s from %s started",
  "error.no_directory": "Directory %s does not exist",
  "log.processing_php": "Processing PHP file: %s"
}
```

**Important**:

- Keep format specifiers (`%s`, `%v`) in the same position
- Don't translate keys (left side), only values (right side)
- Maintain JSON syntax validity

## Code Contributions

### Development Setup

1. **Fork and clone**:

   ```bash
   git clone https://github.com/your-username/wp_plugin_release.git
   cd wp_plugin_release
   ```

### Making Changes

1. **Create a feature branch**:

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following our coding standards:
   - Use `t()` function for all user-facing strings
   - Add appropriate translation keys to both `locales/en.json` and `locales/de.json`
   - Follow Go conventions and existing code style
   - Add tests for new functionality

3. **Test your changes**:

   ```bash
   go test -v
   ```

4. **Run linters**:

   ```bash
   go vet -v
   ```

### Code Standards

#### Internationalization

- **All user-facing strings** must use the `t()` function:

  ```go
  // Good
  logAndPrint(t("log.processing_php", phpFilePath))
  
  // Bad
  logAndPrint("Processing PHP file: " + phpFilePath)
  ```

- **Add translation keys** to both `locales/en.json` and `locales/de.json`

- **Use descriptive key names** with categories:

  ```text
  app.* - Application messages
  error.* - Error messages
  log.* - Log messages
  config.* - Configuration messages
  ```

#### Go Code Style

- Follow standard Go formatting (`go fmt`)
- Use descriptive variable names
- Add comments for exported functions
- Handle errors appropriately
- Use structured logging where possible

### Testing

#### Unit Tests

```bash
make test
```

#### Integration Tests

```bash
# Test with actual plugin directory
./bin/wp_plugin_release /path/to/test/plugin
```

#### Translation Tests

```bash
make test-i18n
```

### Submitting Changes

1. **Commit your changes**:

   ```bash
   git add .
   git commit -m "feat: add support for French translation"
   ```

2. **Push to your fork**:

   ```bash
   git push origin feature/your-feature-name
   ```

3. **Create a pull request** with:
   - Clear description of changes
   - Screenshots if UI changes
   - Translation test results if applicable
   - Reference to any related issues

## Commit Message Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `i18n`: Internationalization changes

### Examples

```text
feat(i18n): add French translation support
fix(config): handle missing config file gracefully
docs: update installation instructions
i18n(de): improve German error messages
```

## 🐛 Bug Reports

When reporting bugs, please include:

1. **Environment information**:
   - Operating system
   - Go version
   - Language/locale settings

2. **Steps to reproduce**

3. **Expected vs actual behavior**

4. **Error messages** (in original language if possible)

5. **Config file** (sanitized, remove sensitive data)

## Feature Requests

For feature requests:

1. **Check existing issues** first
2. **Describe the use case** clearly
3. **Consider internationalization** impact
4. **Provide examples** if applicable

## Development Workflow

### Branch Strategy

- `main` - stable releases
- `develop` - development branch
- `feature/*` - feature branches
- `fix/*` - bug fix branches
- `i18n/*` - translation branches

### Release Process

1. Development happens on `develop`
2. Features merged via pull requests
3. Release candidates tagged as `v1.0.0-rc1`
4. Final releases tagged as `v1.0.0`
5. Automated builds create binaries for all platforms

## Checklist for Contributors

### For Code Changes

- [ ] Code follows project conventions
- [ ] All user-facing strings use `t()` function
- [ ] Translation keys added to all language files
- [ ] Tests pass (`make test`)
- [ ] Linters pass (`make lint`)
- [ ] Translation validation passes (`make i18n-validate`)
- [ ] Documentation updated if needed

### For Translation Changes

- [ ] Translation file is valid JSON
- [ ] All keys from English version are translated
- [ ] Format specifiers preserved correctly
- [ ] Tested with `LANG=<code> wp_plugin_release --help`
- [ ] Cultural appropriateness considered

## Community

- **Be respectful** and value everybody
- **Help others** learn and contribute
- **Provide constructive feedback**
- **Celebrate diversity** in languages and cultures

## Getting Help

- **Issues**: GitHub Issues for bugs and feature requests
- **Discussions**: GitHub Discussions for questions
- **Email**: [security@vaya-consulting.de] for security issues

## License

By contributing, you agree that your contributions will be licensed under the same modified MIT license as the project.

---

Thank you for contributing to wp_plugin_release!
