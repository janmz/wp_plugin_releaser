package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeTestFile writes content to dir/name and fails the test on error.
func writeTestFile(ts *testing.T, dir, name, content string) {
	ts.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
		ts.Fatal(err)
	}
}

func TestFindMainPHPFile(ts *testing.T) {
	dir := ts.TempDir()
	writeTestFile(ts, dir, "a-helper.php", "<?php\n// helper\n")
	writeTestFile(ts, dir, "my-plugin.php", "<?php\n/*\n * Plugin Name: My Plugin\n */\n")
	if got := findMainPHPFile(dir); got != "my-plugin.php" {
		ts.Fatalf("findMainPHPFile = %q, want my-plugin.php", got)
	}
	if got := findMainPHPFile(ts.TempDir()); got != "" {
		ts.Fatalf("findMainPHPFile on empty dir = %q, want empty", got)
	}
}

func TestAddMissingReleaseBuildFields(ts *testing.T) {
	in := []byte("{\n\t\"ssh_host\": \"h\\\"x\",\n\t\"skip_pattern\": [\"a\\\\b\"]\n}\n")
	out, added, err := addMissingReleaseBuildFields(in, "pä.php")
	if err != nil {
		ts.Fatal(err)
	}
	if len(added) != 3 {
		ts.Fatalf("added = %v, want 3 fields", added)
	}
	var cfg ConfigType
	if err := json.Unmarshal(out, &cfg); err != nil {
		ts.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if cfg.Type != "wordpress" || cfg.Version != 2 || cfg.MainPHPFile != "pä.php" {
		ts.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.SSHHost != "h\"x" || len(cfg.SkipPattern) != 1 || cfg.SkipPattern[0] != "a\\b" {
		ts.Fatalf("existing fields changed: %+v", cfg)
	}

	// Complete config stays unchanged.
	again, added2, err := addMissingReleaseBuildFields(out, "other.php")
	if err != nil || added2 != nil || string(again) != string(out) {
		ts.Fatalf("complete config changed: added=%v err=%v", added2, err)
	}

	// Empty object must not get a trailing comma.
	empty, _, err := addMissingReleaseBuildFields([]byte("{}"), "")
	if err != nil || !json.Valid(empty) {
		ts.Fatalf("empty object: err=%v\n%s", err, empty)
	}
}

func TestEnsureReleaseBuildConfig_Create(ts *testing.T) {
	dir := ts.TempDir()
	writeTestFile(ts, dir, "plugin.php", "<?php\n/* Plugin Name: X */\n")
	created, _, err := ensureReleaseBuildConfig(dir)
	if err != nil || !created {
		ts.Fatalf("created=%v err=%v", created, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "update.config"))
	if err != nil {
		ts.Fatal(err)
	}
	var cfg ConfigType
	if err := json.Unmarshal(data, &cfg); err != nil {
		ts.Fatal(err)
	}
	if cfg.Type != "wordpress" || cfg.Version != 2 || cfg.MainPHPFile != "plugin.php" {
		ts.Fatalf("unexpected config: %+v", cfg)
	}
	created, added, err := ensureReleaseBuildConfig(dir)
	if err != nil || created || added != nil {
		ts.Fatalf("second call: created=%v added=%v err=%v", created, added, err)
	}
}
