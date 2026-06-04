package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPIDInfoAcceptsLegacyPIDFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "localtun.pid")
	if err := os.WriteFile(path, []byte("12345\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := readPIDInfo(path)
	if err != nil {
		t.Fatalf("readPIDInfo() error = %v", err)
	}
	if info.PID != 12345 {
		t.Fatalf("PID = %d, want 12345", info.PID)
	}
}

func TestReadPIDInfoAcceptsJSONPIDFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "localtun.pid")
	data := []byte(`{"pid":12345,"executable":"localtun"}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	info, err := readPIDInfo(path)
	if err != nil {
		t.Fatalf("readPIDInfo() error = %v", err)
	}
	if info.PID != 12345 || info.Executable != "localtun" {
		t.Fatalf("info = %+v, want PID 12345 and executable localtun", info)
	}
}

func TestReadPIDInfoRejectsInvalidPIDFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "localtun.pid")
	if err := os.WriteFile(path, []byte("not-a-pid"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := readPIDInfo(path); err != errInvalidPID {
		t.Fatalf("readPIDInfo() error = %v, want errInvalidPID", err)
	}
}
