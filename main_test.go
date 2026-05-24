package main

import (
	"flag"
	"os"
	"testing"
)

func TestParseConfig_Defaults(t *testing.T) {
	wd, _ := os.Getwd()
	cfg, err := parseConfig([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Dir != wd {
		t.Errorf("expected Dir=%q, got %q", wd, cfg.Dir)
	}
	if cfg.Jobs != 5 {
		t.Errorf("expected Jobs=5, got %d", cfg.Jobs)
	}
	if cfg.Monochrome || cfg.AutoYes || cfg.AutoNo || cfg.NoIgnores {
		t.Error("expected all bool flags false by default")
	}
}

func TestParseConfig_ShortFlags(t *testing.T) {
	cfg, err := parseConfig([]string{"-y", "-m", "-j", "4"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AutoYes {
		t.Error("expected AutoYes=true")
	}
	if !cfg.Monochrome {
		t.Error("expected Monochrome=true")
	}
	if cfg.Jobs != 4 {
		t.Errorf("expected Jobs=4, got %d", cfg.Jobs)
	}
}

func TestParseConfig_LongFlags(t *testing.T) {
	cfg, err := parseConfig([]string{"--yes", "--monochrome", "--jobs", "4"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AutoYes {
		t.Error("expected AutoYes=true")
	}
	if !cfg.Monochrome {
		t.Error("expected Monochrome=true")
	}
	if cfg.Jobs != 4 {
		t.Errorf("expected Jobs=4, got %d", cfg.Jobs)
	}
}

func TestParseConfig_DirectoryArg(t *testing.T) {
	cfg, err := parseConfig([]string{"/some/path"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Dir != "/some/path" {
		t.Errorf("expected Dir=/some/path, got %q", cfg.Dir)
	}
}

func TestParseConfig_Conflict(t *testing.T) {
	_, err := parseConfig([]string{"-y", "-n"})
	if err == nil {
		t.Fatal("expected error for -y -n conflict")
	}
}

func TestParseConfig_BadJobsValue(t *testing.T) {
	_, err := parseConfig([]string{"--jobs", "0"})
	if err == nil {
		t.Fatal("expected error for --jobs 0")
	}
}

func TestParseConfig_Version(t *testing.T) {
	_, err := parseConfig([]string{"--version"})
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp for --version, got %v", err)
	}
}

func TestParseConfig_Help(t *testing.T) {
	_, err := parseConfig([]string{"-help"})
	if err != flag.ErrHelp {
		t.Errorf("expected flag.ErrHelp for -help, got %v", err)
	}
}

func TestParseConfig_NoIgnores(t *testing.T) {
	cfg, err := parseConfig([]string{"--no-ignores"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.NoIgnores {
		t.Error("expected NoIgnores=true")
	}
}
