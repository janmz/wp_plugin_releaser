package main

import (
	"bufio"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func changelogPathForWorkDir(workDir string) string {
	candidates := []string{
		filepath.Join(workDir, "Changelog.md"),
		filepath.Join(workDir, "CHANGELOG.md"),
		filepath.Join(workDir, "changelog.md"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return candidates[0]
}

func normalizeChangelogBullets(note string) string {
	note = strings.ReplaceAll(note, "\r\n", "\n")
	note = strings.ReplaceAll(note, "\r", "\n")
	note = strings.TrimSpace(note)
	if note == "" {
		return ""
	}

	lines := strings.Split(note, "\n")
	hasBullets := false
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "- ") {
			hasBullets = true
			break
		}
	}
	if hasBullets {
		return note
	}

	var b strings.Builder
	b.WriteString("- ")
	b.WriteString(strings.TrimSpace(lines[0]))
	for _, l := range lines[1:] {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		b.WriteString("\n  ")
		b.WriteString(t)
	}
	return b.String()
}

func findVersionSectionRange(markdown string, version string) (start int, end int, headerLineEnd int, ok bool) {
	versionPattern := fmt.Sprintf(`(?im)^##\s*(?:\[%s\]|%s)(?:\s|$)`, regexp.QuoteMeta(version), regexp.QuoteMeta(version))
	reStart := regexp.MustCompile(versionPattern)
	m := reStart.FindStringIndex(markdown)
	if m == nil {
		return 0, 0, 0, false
	}
	start = m[0]

	headerLineEnd = strings.Index(markdown[start:], "\n")
	if headerLineEnd < 0 {
		headerLineEnd = len(markdown)
	} else {
		headerLineEnd = start + headerLineEnd + 1
	}

	reNext := regexp.MustCompile(`(?m)^##\s+`)
	nextMatches := reNext.FindAllStringIndex(markdown, -1)
	end = len(markdown)
	for _, nm := range nextMatches {
		if nm[0] > start {
			end = nm[0]
			break
		}
	}
	return start, end, headerLineEnd, true
}

func readChangelog(workDir string, version string) (string, error) {
	changelogPath := changelogPathForWorkDir(workDir)
	if _, err := os.Stat(changelogPath); os.IsNotExist(err) {
		return "", nil
	}

	content, err := os.ReadFile(changelogPath)
	if err != nil {
		return "", err
	}

	contentStr := string(content)
	sectionStart, sectionEnd, _, ok := findVersionSectionRange(contentStr, version)
	if !ok {
		return "", nil
	}

	sectionContent := contentStr[sectionStart:sectionEnd]
	newlineIndex := strings.Index(sectionContent, "\n")
	if newlineIndex >= 0 {
		sectionContent = sectionContent[newlineIndex+1:]
	}

	return strings.TrimSpace(sectionContent), nil
}

func writeChangelog(workDir string, version string, content string) error {
	changelogPath := changelogPathForWorkDir(workDir)
	currentDate := time.Now().Format("2006-01-02")

	var existingContent string
	var newContent string

	bullets := normalizeChangelogBullets(content)
	if bullets == "" {
		return nil
	}

	if _, err := os.Stat(changelogPath); os.IsNotExist(err) {
		newContent = fmt.Sprintf("# Changelog\n\n## [%s] - %s\n\n%s\n", version, currentDate, bullets)
	} else {
		data, err := os.ReadFile(changelogPath)
		if err != nil {
			return err
		}
		existingContent = string(data)

		if _, _, headerEnd, ok := findVersionSectionRange(existingContent, version); ok {
			insertPos := headerEnd
			for insertPos < len(existingContent) {
				if existingContent[insertPos] == '\n' || existingContent[insertPos] == '\r' {
					insertPos++
					continue
				}
				lineEnd := strings.Index(existingContent[insertPos:], "\n")
				if lineEnd < 0 {
					lineEnd = len(existingContent) - insertPos
				}
				line := existingContent[insertPos : insertPos+lineEnd]
				if strings.TrimSpace(line) == "" {
					insertPos += lineEnd
					if insertPos < len(existingContent) && existingContent[insertPos] == '\n' {
						insertPos++
					}
					continue
				}
				break
			}

			prefix := existingContent[:insertPos]
			suffix := existingContent[insertPos:]
			bulletBlock := strings.TrimRight(bullets, "\n") + "\n"
			newContent = prefix + bulletBlock + suffix
		} else {
			changelogHeaderRegex := regexp.MustCompile(`(?im)^#\s*Changelog\s*\n`)
			headerMatch := changelogHeaderRegex.FindStringIndex(existingContent)
			if headerMatch != nil {
				headerEnd := strings.Index(existingContent[headerMatch[1]:], "\n")
				if headerEnd >= 0 {
					headerEnd += headerMatch[1] + 1
				} else {
					headerEnd = headerMatch[1]
				}
				newContent = existingContent[:headerEnd] +
					fmt.Sprintf("\n## [%s] - %s\n\n%s\n\n", version, currentDate, bullets) +
					existingContent[headerEnd:]
			} else {
				newContent = fmt.Sprintf("# Changelog\n\n## [%s] - %s\n\n%s\n\n%s", version, currentDate, bullets, existingContent)
			}
		}
	}

	newContent = strings.ReplaceAll(newContent, "\r\n", "\n")
	newContent = strings.TrimRight(newContent, "\n") + "\n\n"
	return os.WriteFile(changelogPath, []byte(newContent), 0644)
}

func getChangedFiles(workDir string) ([]string, error) {
	if _, err := os.Stat(filepath.Join(workDir, ".git")); os.IsNotExist(err) {
		return []string{}, nil
	}

	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	cmd.Dir = workDir
	_, err := cmd.Output()
	if true || err != nil {
		cmd = exec.Command("git", "diff", "--name-only", "HEAD")
		cmd.Dir = workDir
		output, err := cmd.Output()
		if err != nil {
			return []string{}, nil
		}
		files := strings.Split(strings.TrimSpace(string(output)), "\n")
		var result []string
		for _, file := range files {
			if file != "" {
				result = append(result, file)
			}
		}
		return result, nil
	}
	return []string{}, nil
}

func isInteractiveTerminal() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func promptChangelogText(version string, existingText string, changedFiles []string, textOverride string) (string, error) {
	if strings.TrimSpace(textOverride) != "" {
		return strings.TrimSpace(textOverride), nil
	}
	var preview strings.Builder

	if existingText != "" {
		preview.WriteString(existingText)
		preview.WriteString("\n\n")
	}

	if len(changedFiles) > 0 {
		preview.WriteString("Changed files:\n")
		for _, file := range changedFiles {
			preview.WriteString(fmt.Sprintf("- %s\n", file))
		}
	}

	if preview.Len() > 0 {
		fmt.Println(t("prompt.changelog_preview"))
		fmt.Println(preview.String())
	}

	if os.Getenv("SKIP_CHANGELOG_INPUT") != "" || os.Getenv("AUTO_CHANGELOG") != "" {
		if preview.Len() > 0 {
			logAndPrint("Using auto-generated changelog (SKIP_CHANGELOG_INPUT or AUTO_CHANGELOG is set)")
			return strings.TrimSpace(preview.String()), nil
		}
		return "", nil
	}

	if !isInteractiveTerminal() {
		if preview.Len() > 0 {
			logAndPrint("Non-interactive terminal detected, using auto-generated changelog")
			return strings.TrimSpace(preview.String()), nil
		}
		logAndPrint("Non-interactive terminal detected and no preview available, skipping changelog input")
		return "", nil
	}

	fmt.Print(t("prompt.changelog_text", version))
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		if preview.Len() > 0 {
			logAndPrint("Error reading input, using auto-generated changelog")
			return strings.TrimSpace(preview.String()), nil
		}
		return "", err
	}

	text := strings.TrimSpace(input)
	if text == "" && preview.Len() > 0 {
		return strings.TrimSpace(preview.String()), nil
	}

	return text, nil
}

func processChangelog(workDir string, version string, updateInfo *UpdateInfo, textOverride string) (string, error) {
	logAndPrint(t("log.changelog_reading", version))

	existingText, err := readChangelog(workDir, version)
	if err != nil {
		logAndPrint(t("error.changelog_read", err))
	}

	changedFiles, err := getChangedFiles(workDir)
	if err != nil {
		logAndPrint(t("error.changed_files", err))
		changedFiles = []string{}
	} else {
		logAndPrint(t("log.changed_files_detected", len(changedFiles)))
	}

	changelogText, err := promptChangelogText(version, existingText, changedFiles, textOverride)
	if err != nil {
		return "", fmt.Errorf("%s", t("error.changelog_prompt", err))
	}
	if changelogText == "" {
		return "", nil
	}

	logAndPrint(t("log.changelog_writing", version))
	err = writeChangelog(workDir, version, changelogText)
	if err != nil {
		return "", err
	}
	logAndPrint(t("log.changelog_updated"))

	return changelogText, nil
}

func escapeHTMLInline(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	return html.EscapeString(s)
}

func buildChangelogDLFromFile(workDir string, maxEntries int) (string, error) {
	if maxEntries <= 0 {
		return "", nil
	}
	p := changelogPathForWorkDir(workDir)
	data, err := os.ReadFile(p) // # nosec G304
	if err != nil {
		return "", err
	}
	text := string(data)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")
	type entry struct {
		header string
		lines  []string
	}
	var entries []entry
	var cur *entry
	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t")
		if strings.HasPrefix(line, "## ") {
			h := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			entries = append(entries, entry{header: h})
			cur = &entries[len(entries)-1]
			if len(entries) >= maxEntries {
				continue
			}
			continue
		}
		if cur == nil {
			continue
		}
		if len(entries) > maxEntries {
			continue
		}
		cur.lines = append(cur.lines, line)
	}
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}

	var b strings.Builder
	b.WriteString("<dl>")
	for _, e := range entries {
		header := escapeHTMLInline(e.header)
		if header == "" {
			continue
		}
		b.WriteString("<dt>")
		b.WriteString(header)
		b.WriteString("</dt><dd>")

		var items []string
		var curItem string
		flush := func() {
			curItem = strings.TrimSpace(curItem)
			if curItem != "" {
				items = append(items, curItem)
			}
			curItem = ""
		}
		for _, l := range e.lines {
			t := strings.TrimSpace(l)
			if t == "" {
				continue
			}
			if strings.HasPrefix(t, "- ") {
				flush()
				curItem = strings.TrimSpace(strings.TrimPrefix(t, "- "))
				continue
			}
			if curItem != "" {
				curItem += " " + t
				continue
			}
			items = append(items, t)
		}
		flush()

		if len(items) > 0 {
			b.WriteString("<ul>")
			for _, it := range items {
				b.WriteString("<li>")
				b.WriteString(escapeHTMLInline(it))
				b.WriteString("</li>")
			}
			b.WriteString("</ul>")
		}
		b.WriteString("</dd>")
	}
	b.WriteString("</dl>")

	out := b.String()
	out = strings.ReplaceAll(out, "\n", "")
	out = strings.ReplaceAll(out, "\r", "")
	return out, nil
}

func updateChangelogInUpdateInfo(workDir string, updateInfo *UpdateInfo, changelogText string) {
	if updateInfo.Sections == nil {
		updateInfo.Sections = make(map[string]string)
	}

	htmlText, err := buildChangelogDLFromFile(workDir, 5)
	if err != nil || strings.TrimSpace(htmlText) == "" {
		htmlText = html.EscapeString(changelogText)
		htmlText = strings.ReplaceAll(htmlText, "\n\n", "</p><p>")
		htmlText = strings.ReplaceAll(htmlText, "\n", "<br/>")
		htmlText = "<p>" + htmlText + "</p>"
		htmlText = regexp.MustCompile(`<p></p>`).ReplaceAllString(htmlText, "")
		htmlText = regexp.MustCompile(`- (.+)`).ReplaceAllString(htmlText, "<li>$1</li>")
		htmlText = strings.ReplaceAll(htmlText, "<p><li>", "<ul><li>")
		htmlText = strings.ReplaceAll(htmlText, "</li><br/>", "</li></ul><br/>")
		htmlText = strings.ReplaceAll(htmlText, "\r\n", "")
		htmlText = strings.ReplaceAll(htmlText, "\n", "")
		htmlText = strings.ReplaceAll(htmlText, "\r", "")
	}

	updateInfo.Sections["changelog"] = htmlText
	logAndPrint(t("log.changelog_in_update_info"))
}

