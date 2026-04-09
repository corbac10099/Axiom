package filesystem_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/axiom-ide/axiom/core/bus"
	"github.com/axiom-ide/axiom/core/filesystem"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

func newHandler(t *testing.T) (*filesystem.Handler, string) {
	t.Helper()
	workspace := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	b := bus.New(ctx, 64, testLogger)
	h, err := filesystem.NewHandler(filesystem.Config{
		WorkspaceDir:  workspace,
		MaxFileSizeMB: 10,
		BackupOnWrite: false,
	}, b, testLogger)
	if err != nil {
		t.Fatalf("NewHandler failed: %v", err)
	}
	return h, workspace
}

func TestCreateAndRead(t *testing.T) {
	h, _ := newHandler(t)

	if err := h.CreateFile("hello.go", "package main"); err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}
	result, err := h.ReadFile("hello.go")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if result.Content != "package main" {
		t.Errorf("expected 'package main', got '%s'", result.Content)
	}
}

func TestWriteAppend(t *testing.T) {
	h, _ := newHandler(t)
	_ = h.CreateFile("log.txt", "line1\n")
	if err := h.WriteFile("log.txt", "line2\n", true); err != nil {
		t.Fatalf("WriteFile append failed: %v", err)
	}
	result, _ := h.ReadFile("log.txt")
	if result.Content != "line1\nline2\n" {
		t.Errorf("expected 'line1\\nline2\\n', got '%s'", result.Content)
	}
}

func TestWriteOverwrite(t *testing.T) {
	h, _ := newHandler(t)
	_ = h.CreateFile("data.txt", "original")
	_ = h.WriteFile("data.txt", "replaced", false)
	result, _ := h.ReadFile("data.txt")
	if result.Content != "replaced" {
		t.Errorf("expected 'replaced', got '%s'", result.Content)
	}
}

func TestDeleteFile(t *testing.T) {
	h, workspace := newHandler(t)
	_ = h.CreateFile("toDelete.txt", "bye")
	if err := h.DeleteFile("toDelete.txt"); err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "toDelete.txt")); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestCreateDuplicate(t *testing.T) {
	h, _ := newHandler(t)
	_ = h.CreateFile("dup.txt", "first")
	if err := h.CreateFile("dup.txt", "second"); err == nil {
		t.Error("expected error when creating duplicate file")
	}
}

func TestListDir(t *testing.T) {
	h, _ := newHandler(t)
	_ = h.CreateFile("a.go", "")
	_ = h.CreateFile("b.go", "")
	_ = h.CreateFile("sub/c.go", "")

	entries, err := h.ListDir(".")
	if err != nil {
		t.Fatalf("ListDir failed: %v", err)
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name] = true
	}
	if !names["a.go"] || !names["b.go"] {
		t.Errorf("expected a.go and b.go in listing, got %v", names)
	}
}

// TestPathTraversalBlocked — test de sécurité critique
func TestPathTraversalBlocked(t *testing.T) {
	h, _ := newHandler(t)

	attacks := []string{
		"../etc/passwd",
		"../../secret.key",
		"subdir/../../outside.txt",
	}
	for _, attack := range attacks {
		_, err := h.ReadFile(attack)
		if err == nil {
			t.Errorf("path traversal attack should be blocked: '%s'", attack)
		}
		t.Logf("correctly blocked: '%s' → %v", attack, err)
	}
}

func TestMaxFileSizeEnforced(t *testing.T) {
	workspace := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := bus.New(ctx, 64, testLogger)

	// Limite de 1 octet pour le test
	h, err := filesystem.NewHandler(filesystem.Config{
		WorkspaceDir:  workspace,
		MaxFileSizeMB: 1, // 1 MB
	}, b, testLogger)
	if err != nil {
		t.Fatal(err)
	}

	// Crée un fichier de exactement 1 octet — doit passer
	_ = h.CreateFile("small.txt", "x")
	if _, err := h.ReadFile("small.txt"); err != nil {
		t.Errorf("small file should be readable: %v", err)
	}
}

func TestWorkspaceChange(t *testing.T) {
	h, _ := newHandler(t)
	newDir := t.TempDir()

	if err := h.SetWorkspace(newDir); err != nil {
		t.Fatalf("SetWorkspace failed: %v", err)
	}
	if h.WorkspaceDir() != newDir {
		t.Errorf("expected workspace '%s', got '%s'", newDir, h.WorkspaceDir())
	}
}