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

	// Skip rename detection for more than one file, which makes git write a
	// warning to stderr.
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "diff.renameLimit")
	t.Setenv("GIT_CONFIG_VALUE_0", "1")

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
	write("a1.md", "one")
	write("a2.md", "two")
	run("add", ".")
	run("commit", "-q", "-m", "initial")
	since := run("rev-parse", "HEAD")

	// Changed in two commits, added and removed, with a non-ASCII name, and
	// moved and changed.
	write("edited.md", "new")
	write("removed.md", "temporary")
	write("content/posts/café/index.md", "café")
	run("mv", "a1.md", "b1.md")
	run("mv", "a2.md", "b2.md")
	write("b1.md", "one!")
	write("b2.md", "two!")
	run("add", ".")
	run("commit", "-q", "-m", "first")

	write("edited.md", "newer")
	run("rm", "-q", "removed.md")
	run("commit", "-q", "-a", "-m", "second")

	files, err := (&git{dir: dir}).changedFiles(since)
	require.NoError(t, err)
	assert.ElementsMatch(t, []ModifiedFile{
		{Filename: "a1.md", Content: "one"},
		{Filename: "a2.md", Content: "two"},
		{Filename: "b1.md", Content: ""},
		{Filename: "b2.md", Content: ""},
		{Filename: "content/posts/café/index.md", Content: ""},
		{Filename: "edited.md", Content: "old"},
	}, files)
}
