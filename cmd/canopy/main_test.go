package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindRepoRoot_DirectGitDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := findRepoRoot(root)
	assert.Equal(t, root, got)
}

func TestFindRepoRoot_NestedSubdirectory(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "sub", "deep")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	got := findRepoRoot(deep)
	assert.Equal(t, root, got)
}

func TestFindRepoRoot_NoGitAncestor(t *testing.T) {
	t.Parallel()
	// TempDir has no .git directory anywhere in its ancestry
	// (unless /tmp itself is a repo, which would be unusual).
	dir := t.TempDir()

	got := findRepoRoot(dir)
	assert.Equal(t, dir, got)
}
