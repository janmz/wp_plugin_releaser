package main

import (
	"archive/zip"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func init() {
	logger = log.New(io.Discard, "", 0)
}
func TestGetHigherVersion(ts *testing.T) {
	cases := []struct{ a, b, want string }{
		{"", "", ""},
		{"1.0.0", "", "1.0.0"},
		{"", "1.2.3", "1.2.3"},
		{"1.0.0", "1.0.0", "1.0.0"},
		{"1.0.1", "1.0.0", "1.0.1"},
		{"1.2.0", "1.10.0", "1.10.0"},
		{"2.0", "1.9.9", "2.0"},
		{"1.2.3", "1.2.3.4", "1.2.3.4"},
		{"1.2.3.4", "1.2.3", "1.2.3.4"},
	}
	for _, c := range cases {
		if got := getHigherVersion(c.a, c.b); got != c.want {
			ts.Fatalf("getHigherVersion(%q,%q)=%q want %q", c.a, c.b, got, c.want)
		}
	}
}

func TestMarshalWithoutHTMLescaping(ts *testing.T) {
	input := map[string]string{"url": "https://example.com?a=1&b=2"}
	data, err := marshalWithoutHTMLescaping(input)
	if err != nil {
		ts.Fatalf("marshalWithoutHTMLescaping error: %v", err)
	}
	s := string(data)
	if strings.Contains(s, "\\u003c") || strings.Contains(s, "\\u003e") || strings.Contains(s, "\\u0026") {
		ts.Fatalf("HTML escaping detected in output: %s", s)
	}
	// ensure pretty print with newline at end
	if !strings.HasSuffix(s, "\n") {
		ts.Fatalf("expected trailing newline for pretty JSON, got: %q", s)
	}
}

func writeFile(ts *testing.T, path, content string) {
	ts.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		ts.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		ts.Fatalf("write file: %v", err)
	}
}

func TestUpdateInfoReadProcessWrite(ts *testing.T) {
	dir := ts.TempDir()
	uiPath := filepath.Join(dir, "Updates", "update_info.json")
	// initial content has older version and extra unknown field
	initial := map[string]any{
		"version":      "1.0.0",
		"last_updated": "2024-01-01 00:00:00",
		"download_url": "https://example.com/plugin/plugin-v1.0.0.zip",
		"unknown":      map[string]any{"keep": true},
	}
	raw, _ := json.Marshal(initial)
	writeFile(ts, uiPath, string(raw))

	ui, all, err := getUpdateInfo(uiPath)
	if err != nil {
		ts.Fatalf("getUpdateInfo error: %v", err)
	}
	if ui.Version != "1.0.0" || all["unknown"] == nil {
		ts.Fatalf("unexpected getUpdateInfo results: ui=%+v all=%+v", ui, all)
	}

	if err := processUpdateInfo(ui, "1.2.0"); err != nil {
		ts.Fatalf("processUpdateInfo error: %v", err)
	}
	if ui.Version != "1.2.0" {
		ts.Fatalf("version not updated, got %q", ui.Version)
	}

	// setUpdateInfo should merge unknown fields and create a backup
	if err := setUpdateInfo(ui, all, uiPath); err != nil {
		ts.Fatalf("setUpdateInfo error: %v", err)
	}
	// backup exists
	if _, err := os.Stat(uiPath + ".bak"); err != nil {
		ts.Fatalf("expected backup file: %v", err)
	}
	// read back and ensure unknown kept and version bumped
	data, err := os.ReadFile(uiPath)
	if err != nil {
		ts.Fatalf("read back: %v", err)
	}
	var final map[string]any
	if err := json.Unmarshal(data, &final); err != nil {
		ts.Fatalf("unmarshal back: %v", err)
	}
	if final["version"] != "1.2.0" {
		ts.Fatalf("final version mismatch: %v", final["version"])
	}
	if _, ok := final["unknown"].(map[string]any); !ok {
		ts.Fatalf("unknown field lost: %v", final)
	}
}

func TestShouldSkipAndZipCreation(ts *testing.T) {
	dir := ts.TempDir()
	// create files
	writeFile(ts, filepath.Join(dir, "a.txt"), "A")
	writeFile(ts, filepath.Join(dir, "b.log"), "B")
	writeFile(ts, filepath.Join(dir, "Updates", "ignored.txt"), "C")
	writeFile(ts, filepath.Join(dir, "nested", "Thumbs.db"), "D")

	zipPath := filepath.Join(dir, "Updates", "out.zip")
	// custom skip to ignore .log files
	if err := createZipFile(dir, zipPath, []string{"*.log"}, "slug"); err != nil {
		ts.Fatalf("createZipFile error: %v", err)
	}

	// open the zip and assert contents
	f, err := os.Open(zipPath)
	if err != nil {
		ts.Fatalf("open zip: %v", err)
	}
	defer f.Close()
	stat, _ := f.Stat()
	zr, err := zip.NewReader(f, stat.Size())
	if err != nil {
		ts.Fatalf("zip reader: %v", err)
	}
	var names []string
	for _, zf := range zr.File {
		names = append(names, zf.Name)
	}
	joined := strings.Join(names, "|")
	if strings.Contains(joined, "Updates/") || strings.Contains(joined, "Thumbs.db") || strings.Contains(joined, ".log") {
		ts.Fatalf("zip contains skipped items: %v", names)
	}
	// should include a.txt under slug/
	want := "slug/a.txt"
	found := false
	for _, n := range names {
		if n == want {
			found = true
			break
		}
	}
	if !found {
		ts.Fatalf("expected file %q in zip, got %v", want, names)
	}
}

func TestParseRemotePath(ts *testing.T) {
	cases := []struct {
		url     string
		base    string
		want    string
		wantErr bool
	}{
		{"https://example.com/a/b/file.zip", "/var/www", "/var/www/a/b", false},
		{"https://example.com/file.zip", "/var/www/", "/var/www", false},
		{"https://example.com/a/b/", "/var/www", "", true},
		{"invalid://://", "/var/www", "", true},
	}
	for _, c := range cases {
		got, err := parseRemotePath(c.url, c.base)
		if c.wantErr {
			if err == nil {
				ts.Fatalf("expected error for %q", c.url)
			}
			continue
		}
		if err != nil || got != c.want {
			ts.Fatalf("parseRemotePath(%q,%q)=%q,%v want %q,nil", c.url, c.base, got, err, c.want)
		}
	}
}

func TestProcessMainPHPFile_Minimal(ts *testing.T) {
	dir := ts.TempDir()
	// minimal plugin PHP content:
	// - Version in header and class differ
	// - PUC call with URL and slug
	php := `<?php
/*
 * Plugin Name: TestPlugin
 * Version: 1.0.0
 */
class TestPlugin { private $version = '1.2.0'; }
require 'vendor/autoload.php';
$updateChecker = PucFactory::buildUpdateChecker('https://example.com/updates/plugin-v0.9.0.zip', __FILE__, 'old-slug');
`
	main := "plugin.php"
	writeFile(ts, filepath.Join(dir, main), php)

	ui := &UpdateInfo{DownloadURL: "https://example.com/updates/plugin-v0.9.0.zip", Slug: "new-slug"}
	// ensure log file setup to avoid nil logger writes in tests
	initLogging(dir)
	defer logFile.Close()

	ver, err := processMainPHPFile(dir, main, ui)
	if err != nil {
		ts.Fatalf("processMainPHPFile error: %v", err)
	}
	if ver != "1.2.0" { // highest of 1.0.0 and 1.2.0
		ts.Fatalf("expected detected version 1.2.0, got %q", ver)
	}
	// read file to ensure URL changed to update_info.json and slug updated
	b, err := os.ReadFile(filepath.Join(dir, main))
	if err != nil {
		ts.Fatalf("read back php: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "new-slug") {
		ts.Fatalf("slug not updated in PUC: %s", s)
	}
	if !strings.Contains(s, "'https://example.com/updates/update_info.json'") {
		ts.Fatalf("download URL not updated to update_info.json: %s", s)
	}
	// ensure last-update injected or updated
	if !strings.Contains(strings.ToLower(s), "last-update:") {
		ts.Fatalf("Last-Update not added/updated: %s", s)
	}
}

func TestProcessUpdateInfo_NoChangeWhenNewer(ts *testing.T) {
	ui := &UpdateInfo{Version: "2.0.0"}
	if err := processUpdateInfo(ui, "1.9.9"); err != nil {
		ts.Fatalf("processUpdateInfo: %v", err)
	}
	if ui.Version != "2.0.0" {
		ts.Fatalf("version should remain 2.0.0, got %s", ui.Version)
	}
}

func TestCreateRemoteDirCommandEscapes(t *testing.T) {
	// This function requires a real ssh.Client; instead, test the command format indirectly by ensuring
	// uploadFileViaSFTP uses ToSlash and mkdir -p path formatting. We cannot instantiate ssh.Client here.
	// This test is intentionally a no-op placeholder to document limitation of pure unit tests for SSH.
	_ = time.Now()
}
