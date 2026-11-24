package main

/*
 * releasebuild_config.go: Prepares update.config for the successor tool releaseBuild.
 *
 * releaseBuild (https://github.com/janmz/releaseBuild) replaces wp_plugin_release.
 * It reads the same update.config, but needs "version": 2, "type": "wordpress"
 * and "main_php_file". These fields are created or inserted here so that the
 * project can be switched to releaseBuild without manual editing.
 */

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// releaseBuildType is the project type releaseBuild uses for WordPress plugins.
const releaseBuildType = "wordpress"

// releaseBuildConfigVersion is the update.config schema version of releaseBuild.
const releaseBuildConfigVersion = 2

// pluginHeaderScanLimit is the number of bytes searched for the "Plugin Name:" header.
const pluginHeaderScanLimit = 8192

// findMainPHPFile returns the name of the PHP file in workDir that carries the
// WordPress plugin header ("Plugin Name:"), or "" if there is none.
func findMainPHPFile(workDir string) string {
	matches, err := filepath.Glob(filepath.Join(workDir, "*.php"))
	if err != nil {
		return ""
	}
	sort.Strings(matches)
	for _, m := range matches {
		if hasPluginHeader(m) {
			return filepath.Base(m)
		}
	}
	return ""
}

// hasPluginHeader reports whether the beginning of the file contains "Plugin Name:".
func hasPluginHeader(path string) bool {
	f, err := os.Open(path) // #nosec G304 -- path comes from Glob in the plugin directory
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, pluginHeaderScanLimit)
	n, _ := bufio.NewReader(f).Read(buf)
	return bytes.Contains(buf[:n], []byte("Plugin Name:"))
}

// newReleaseBuildConfig returns the content of a new update.config with all
// fields releaseBuild and wp_plugin_release need; SSH fields stay empty.
func newReleaseBuildConfig(mainPHPFile string) ([]byte, error) {
	cfg := ConfigType{
		Version:     releaseBuildConfigVersion,
		Type:        releaseBuildType,
		MainPHPFile: mainPHPFile,
		SkipPattern: []string{},
		SSHPort:     "22",
	}
	data, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// addMissingReleaseBuildFields inserts the fields "type", "version" and
// "main_php_file" into the JSON object in data if they are missing. All other
// content and the formatting are preserved. It returns the new content and
// the names of the inserted fields (nil if nothing was missing).
func addMissingReleaseBuildFields(data []byte, mainPHPFile string) ([]byte, []string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, nil, err
	}

	var added []string
	var lines []string
	if _, ok := fields["type"]; !ok {
		added = append(added, "type")
		lines = append(lines, fmt.Sprintf("%q: %q", "type", releaseBuildType))
	}
	if _, ok := fields["version"]; !ok {
		added = append(added, "version")
		lines = append(lines, fmt.Sprintf("%q: %d", "version", releaseBuildConfigVersion))
	}
	if _, ok := fields["main_php_file"]; !ok {
		value, err := json.Marshal(mainPHPFile)
		if err != nil {
			return nil, nil, err
		}
		added = append(added, "main_php_file")
		lines = append(lines, fmt.Sprintf("%q: %s", "main_php_file", value))
	}
	if len(added) == 0 {
		return data, nil, nil
	}

	open := bytes.IndexByte(data, '{')
	if open < 0 {
		return nil, nil, fmt.Errorf("no JSON object found")
	}
	insert := "\n\t" + strings.Join(lines, ",\n\t")
	if len(fields) > 0 {
		insert += ","
	} else {
		insert += "\n"
	}
	result := make([]byte, 0, len(data)+len(insert))
	result = append(result, data[:open+1]...)
	result = append(result, insert...)
	result = append(result, data[open+1:]...)
	if !json.Valid(result) {
		return nil, nil, fmt.Errorf("inserting fields produced invalid JSON")
	}
	return result, added, nil
}

// ensureReleaseBuildConfig creates update.config in workDir if it does not
// exist, or adds the fields releaseBuild needs. created is true if the file
// was newly created; added lists the inserted fields.
func ensureReleaseBuildConfig(workDir string) (created bool, added []string, err error) {
	path := filepath.Join(workDir, "update.config")
	data, err := os.ReadFile(path) // #nosec G304 -- fixed file name in the plugin directory
	if os.IsNotExist(err) {
		content, err := newReleaseBuildConfig(findMainPHPFile(workDir))
		if err != nil {
			return false, nil, err
		}
		if err := os.WriteFile(path, content, 0600); err != nil {
			return false, nil, err
		}
		return true, nil, nil
	}
	if err != nil {
		return false, nil, err
	}

	newData, added, err := addMissingReleaseBuildFields(data, findMainPHPFile(workDir))
	if err != nil || len(added) == 0 {
		return false, nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return false, nil, err
	}
	if err := os.WriteFile(path, newData, info.Mode().Perm()); err != nil {
		return false, nil, err
	}
	return false, added, nil
}
