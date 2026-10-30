package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangedFiles(t *testing.T) {
	// Ignore the global and system configurations, such as core.quotePath.
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=Eagle", "-c", "user.email=eagle@eagle"}, args...)...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return strings.TrimSpace(string(out))
	}

	write := func(filename, content string) {
		t.Helper()
		filename = filepath.Join(dir, filename)
		require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o755))
		require.NoError(t, os.WriteFile(filename, []byte(content), 0o644))
	}

	run("init", "-q")
	write("edited.md", "old")
	run("add", ".")
	run("commit", "-q", "-m", "initial")
	since := run("rev-parse", "HEAD")

	// Changed in two commits, added and removed, and with a non-ASCII name.
	write("edited.md", "new")
	write("removed.md", "temporary")
	write("content/posts/café/index.md", "café")
	run("add", ".")
	run("commit", "-q", "-m", "first")

	write("edited.md", "newer")
	run("rm", "-q", "removed.md")
	run("commit", "-q", "-a", "-m", "second")

	files, err := (&git{dir: dir}).changedFiles(since)
	require.NoError(t, err)
	assert.ElementsMatch(t, []ModifiedFile{
		{Filename: "content/posts/café/index.md", Content: ""},
		{Filename: "edited.md", Content: "old"},
	}, files)
}
