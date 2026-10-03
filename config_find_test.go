// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zauberhaus/config"
)

// chdir changes the working directory for the duration of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()

	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))

	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestLoad_FindConfigFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CONFIG", "")

	chdir(t, t.TempDir())

	t.Run("ignores entries that can't be a config file", func(t *testing.T) {
		noise := t.TempDir()

		// None of these may be picked up, although the name matches.
		require.NoError(t, os.Mkdir(filepath.Join(noise, "find-app.json"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(noise, ".find-app.json"), []byte(`{"host": "dotfile"}`), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(noise, "find-app..json"), []byte(`{"host": "dots"}`), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(noise, "find-app"), []byte(`{"host": "no-ext"}`), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(noise, "a.b"), []byte(`{"host": "short"}`), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(noise, "other.json"), []byte(`{"host": "other"}`), 0644))

		real := t.TempDir()
		want := filepath.Join(real, "find-app.json")
		require.NoError(t, os.WriteFile(want, []byte(`{"host": "real.host.com"}`), 0644))

		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("find-app"), config.WithPaths(noise, real))
		require.NoError(t, err)
		assert.Equal(t, want, f)
		assert.Equal(t, "real.host.com", cfg.Host)
	})

	t.Run("ignores files with an unknown extension", func(t *testing.T) {
		noise := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(noise, "find-app.txt"), []byte(`{"host": "txt"}`), 0644))

		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("find-app"), config.WithPaths(noise))
		require.NoError(t, err)
		assert.Empty(t, f) // not picked up, defaults are used
		assert.Equal(t, "localhost", cfg.Host)
	})

	t.Run("skips search paths that can't be read", func(t *testing.T) {
		real := t.TempDir()
		want := filepath.Join(real, "find-app.yaml")
		require.NoError(t, os.WriteFile(want, []byte("host: real.host.com\n"), 0644))

		missing := filepath.Join(t.TempDir(), "does-not-exist")

		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("find-app"), config.WithPaths(missing, real))
		require.NoError(t, err)
		assert.Equal(t, want, f)
		assert.Equal(t, "real.host.com", cfg.Host)
	})

	t.Run("no config file found", func(t *testing.T) {
		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("find-app"), config.WithPaths(t.TempDir()))
		require.NoError(t, err)
		assert.Empty(t, f)
		assert.Equal(t, "localhost", cfg.Host)
	})

	t.Run("error if the home directory is unknown", func(t *testing.T) {
		if runtime.GOOS == "windows" || runtime.GOOS == "plan9" {
			t.Skip("home directory is not taken from $HOME")
		}

		t.Setenv("HOME", "")

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("find-app"))
		assert.Nil(t, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "get homedir failed")
	})

	t.Run("error if the working directory is gone", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("a working directory can't be removed on windows")
		}

		gone := t.TempDir()
		chdir(t, gone)
		require.NoError(t, os.Remove(gone))

		if _, err := os.Getwd(); err == nil {
			t.Skip("os.Getwd still works after removing the working directory")
		}

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("find-app"), config.WithWorkingDir(true))
		assert.Nil(t, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "get current index failed")
	})
}

func TestLoad_OptionalPointerFromJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CONFIG", "")

	chdir(t, t.TempDir())

	file := filepath.Join(t.TempDir(), "sub2.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"sub2": {"name": "json name"}}`), 0644))

	cfg, f, err := config.Load[*TestLoadConfig](config.WithFile(file))
	require.NoError(t, err)
	assert.Equal(t, file, f)

	if assert.NotNil(t, cfg.Sub2) {
		assert.Equal(t, "json name", cfg.Sub2.Name)
		assert.Equal(t, "sub2-default", cfg.Sub2.Other) // default survives the reload
	}
}

func TestLoad_WorkingDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CONFIG", "")

	wd, err := filepath.EvalSymlinks(t.TempDir()) // as reported by os.Getwd
	require.NoError(t, err)
	chdir(t, wd)

	wdFile := filepath.Join(wd, "wd-app.yaml")
	require.NoError(t, os.WriteFile(wdFile, []byte("host: wd.host.com\n"), 0644))

	t.Run("not searched by default", func(t *testing.T) {
		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("wd-app"))
		require.NoError(t, err)
		assert.Empty(t, f)
		assert.Equal(t, "localhost", cfg.Host)
	})

	t.Run("searched when enabled", func(t *testing.T) {
		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("wd-app"), config.WithWorkingDir(true))
		require.NoError(t, err)
		assert.Equal(t, wdFile, f)
		assert.Equal(t, "wd.host.com", cfg.Host)
	})

	t.Run("home directory is still searched", func(t *testing.T) {
		homeFile := filepath.Join(home, "wd-app.json")
		require.NoError(t, os.WriteFile(homeFile, []byte(`{"host": "home.host.com"}`), 0644))
		t.Cleanup(func() { _ = os.Remove(homeFile) })

		cfg, f, err := config.Load[*TestLoadConfig](config.WithName("wd-app"))
		require.NoError(t, err)
		assert.Equal(t, homeFile, f)
		assert.Equal(t, "home.host.com", cfg.Host)

		// the working directory comes before the home directory
		cfg, f, err = config.Load[*TestLoadConfig](config.WithName("wd-app"), config.WithWorkingDir(true))
		require.NoError(t, err)
		assert.Equal(t, wdFile, f)
		assert.Equal(t, "wd.host.com", cfg.Host)
	})
}

func TestLoad_PathTraversal(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CONFIG", "")

	dir := t.TempDir()
	chdir(t, dir)

	t.Run("dots inside a name are fine", func(t *testing.T) {
		file := filepath.Join(dir, "app..v2.json")
		require.NoError(t, os.WriteFile(file, []byte(`{"host": "dots.host.com"}`), 0644))

		cfg, _, err := config.Load[*TestLoadConfig](config.WithFile(file))
		require.NoError(t, err)
		assert.Equal(t, "dots.host.com", cfg.Host)

		t.Setenv("CONFIG", file)

		cfg, _, err = config.Load[*TestLoadConfig]()
		require.NoError(t, err)
		assert.Equal(t, "dots.host.com", cfg.Host)
	})

	for _, p := range []string{"..", "../app.json", "sub/../../app.json", dir + "/../app.json"} {
		t.Run(p, func(t *testing.T) {
			_, _, err := config.Load[*TestLoadConfig](config.WithFile(p))
			assert.ErrorContains(t, err, "path traversal attempt")

			t.Setenv("CONFIG", p)

			_, _, err = config.Load[*TestLoadConfig]()
			assert.ErrorContains(t, err, "path traversal attempt")
		})
	}
}

func TestLoad_FileChecks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes and symlinks work differently on windows")
	}

	t.Setenv("HOME", t.TempDir())
	t.Setenv("CONFIG", "")

	chdir(t, t.TempDir())

	writeFile := func(t *testing.T, dir, name string, mode os.FileMode) string {
		t.Helper()

		file := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(file, []byte(`{"host": "file.host.com"}`), 0600))
		require.NoError(t, os.Chmod(file, mode)) // not affected by the umask

		return file
	}

	t.Run("world-writable file is rejected", func(t *testing.T) {
		dir := t.TempDir()
		file := writeFile(t, dir, "ww-app.json", 0666)

		_, _, err := config.Load[*TestLoadConfig](config.WithFile(file))
		assert.ErrorContains(t, err, "is writable by others")

		_, _, err = config.Load[*TestLoadConfig](config.WithName("ww-app"), config.WithPaths(dir))
		assert.ErrorContains(t, err, "is writable by others")
	})

	t.Run("world-writable file is allowed if enabled", func(t *testing.T) {
		file := writeFile(t, t.TempDir(), "ww-app.json", 0666)

		cfg, _, err := config.Load[*TestLoadConfig](config.WithFile(file), config.WithWorldWritable(true))
		require.NoError(t, err)
		assert.Equal(t, "file.host.com", cfg.Host)
	})

	t.Run("group-writable file is fine", func(t *testing.T) {
		file := writeFile(t, t.TempDir(), "gw-app.json", 0664)

		cfg, _, err := config.Load[*TestLoadConfig](config.WithFile(file))
		require.NoError(t, err)
		assert.Equal(t, "file.host.com", cfg.Host)
	})

	t.Run("FIFO is rejected without blocking", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "fifo-app.json")
		require.NoError(t, syscall.Mkfifo(file, 0600))

		_, _, err := config.Load[*TestLoadConfig](config.WithFile(file))
		assert.ErrorContains(t, err, "is not a regular file")
	})

	t.Run("unreadable file is an error", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root can read any file")
		}

		file := writeFile(t, t.TempDir(), "unreadable-app.json", 0o000)

		cfg, _, err := config.Load[*TestLoadConfig](config.WithFile(file))
		assert.Nil(t, cfg)
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("symlink inside the search directory is followed", func(t *testing.T) {
		dir := t.TempDir()
		target := writeFile(t, dir, "target.json", 0644)
		require.NoError(t, os.Symlink(filepath.Base(target), filepath.Join(dir, "link-app.json")))

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("link-app"), config.WithPaths(dir))
		require.NoError(t, err)
		assert.Equal(t, "file.host.com", cfg.Host)
	})

	t.Run("symlink out of the search directory is rejected", func(t *testing.T) {
		target := writeFile(t, t.TempDir(), "target.json", 0644)

		dir := t.TempDir()
		link := filepath.Join(dir, "link-app.json")
		require.NoError(t, os.Symlink(target, link))

		_, _, err := config.Load[*TestLoadConfig](config.WithName("link-app"), config.WithPaths(dir))
		assert.Error(t, err)

		// an explicitly given file may be a symlink to anywhere
		cfg, _, err := config.Load[*TestLoadConfig](config.WithFile(link))
		require.NoError(t, err)
		assert.Equal(t, "file.host.com", cfg.Host)

		t.Setenv("CONFIG", link)

		cfg, _, err = config.Load[*TestLoadConfig]()
		require.NoError(t, err)
		assert.Equal(t, "file.host.com", cfg.Host)
	})
}
