// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// cspell:words	noctx

package config_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zauberhaus/config"
	"go.uber.org/mock/gomock"
)

// memStorage is an in-memory config.Storage.
type memStorage struct {
	data   map[string]any
	allErr error
	setErr error
}

var _ config.Storage = (*memStorage)(nil)

func newMemStorage(data map[string]any) *memStorage {
	if data == nil {
		data = map[string]any{}
	}

	return &memStorage{data: data}
}

func (m *memStorage) All(_ context.Context) (map[string]any, error) {
	if m.allErr != nil {
		return nil, m.allErr
	}

	return m.data, nil
}

func (m *memStorage) Get(_ context.Context, key string) (any, error) {
	v, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return v, nil
}

func (m *memStorage) Set(_ context.Context, key string, val any) error {
	if m.setErr != nil {
		return m.setErr
	}

	m.data[key] = val

	return nil
}

func (m *memStorage) Delete(_ context.Context, key string) error {
	if m.setErr != nil {
		return m.setErr
	}

	delete(m.data, key)

	return nil
}

func TestStorage_Interface(t *testing.T) {
	ctx := t.Context()

	t.Run("set and get", func(t *testing.T) {
		var s config.Storage = newMemStorage(nil)

		require.NoError(t, s.Set(ctx, "host", "example.com"))

		v, err := s.Get(ctx, "host")
		require.NoError(t, err)
		assert.Equal(t, "example.com", v)
	})

	t.Run("get missing key", func(t *testing.T) {
		var s config.Storage = newMemStorage(nil)

		_, err := s.Get(ctx, "missing")
		assert.Error(t, err)
	})

	t.Run("set overwrites", func(t *testing.T) {
		var s config.Storage = newMemStorage(map[string]any{"port": 1})

		require.NoError(t, s.Set(ctx, "port", 2))

		v, err := s.Get(ctx, "port")
		require.NoError(t, err)
		assert.Equal(t, 2, v)
	})

	t.Run("all returns every entry", func(t *testing.T) {
		var s config.Storage = newMemStorage(nil)

		require.NoError(t, s.Set(ctx, "host", "a"))
		require.NoError(t, s.Set(ctx, "port", 1))

		all, err := s.All(ctx)
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"host": "a", "port": 1}, all)
	})

	t.Run("errors are propagated", func(t *testing.T) {
		m := newMemStorage(nil)
		m.allErr = errors.New("all failed")
		m.setErr = errors.New("set failed")

		_, err := m.All(ctx)
		assert.EqualError(t, err, "all failed")
		assert.EqualError(t, m.Set(ctx, "k", "v"), "set failed")
	})
}

func TestLoad_Storage(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CONFIG", "")

	require.NoError(t, os.Chdir(tempDir))

	t.Run("nil storage is ignored", func(t *testing.T) {
		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("storage-nil"), config.WithStorage(nil))
		require.NoError(t, err)
		assert.Equal(t, "localhost", cfg.Host)
	})

	t.Run("empty storage keeps defaults", func(t *testing.T) {
		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("storage-empty"), config.WithStorage(newMemStorage(nil)))
		require.NoError(t, err)
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 8080, cfg.Port)
		assert.True(t, cfg.Enabled)
	})

	t.Run("values override defaults", func(t *testing.T) {
		s := newMemStorage(map[string]any{
			"host":     "storage.host.com",
			"port":     7070,
			"enabled":  false,
			"sub.name": "storage-sub",
		})

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("storage-defaults"), config.WithStorage(s))
		require.NoError(t, err)

		assert.Equal(t, "storage.host.com", cfg.Host)
		assert.Equal(t, 7070, cfg.Port)
		assert.False(t, cfg.Enabled)
		assert.Equal(t, "storage-sub", cfg.Sub.Name)
	})

	t.Run("values override config file", func(t *testing.T) {
		file := filepath.Join(tempDir, "storage-file.json")
		require.NoError(t, os.WriteFile(file, []byte(`{"host": "file.host.com", "port": 9090}`), 0644))

		s := newMemStorage(map[string]any{"host": "storage.host.com"})

		cfg, f, err := config.Load[*TestLoadConfig](config.WithFile(file), config.WithStorage(s))
		require.NoError(t, err)
		assert.Equal(t, file, f)

		assert.Equal(t, "storage.host.com", cfg.Host) // from storage
		assert.Equal(t, 9090, cfg.Port)               // from file
	})

	t.Run("env overrides storage", func(t *testing.T) {
		t.Setenv("STORAGE_ENV_HOST", "env.host.com")

		s := newMemStorage(map[string]any{"host": "storage.host.com", "port": 7070})

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("storage-env"), config.WithStorage(s))
		require.NoError(t, err)

		assert.Equal(t, "env.host.com", cfg.Host) // from env
		assert.Equal(t, 7070, cfg.Port)           // from storage
	})

	t.Run("optional struct pointer is created", func(t *testing.T) {
		s := newMemStorage(map[string]any{"sub2.name": "storage-name"})

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("storage-ptr"), config.WithStorage(s))
		require.NoError(t, err)

		if assert.NotNil(t, cfg.Sub2) {
			assert.Equal(t, "storage-name", cfg.Sub2.Name)
		}
	})

	t.Run("error from All is returned", func(t *testing.T) {
		s := newMemStorage(nil)
		s.allErr = errors.New("storage unavailable")

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("storage-err"), config.WithStorage(s))
		assert.Nil(t, cfg)
		assert.EqualError(t, err, "storage unavailable")
	})

	t.Run("unknown key is returned as error", func(t *testing.T) {
		s := newMemStorage(map[string]any{"does.not.exist": "x"})

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("storage-unknown"), config.WithStorage(s))
		assert.Nil(t, cfg)
		assert.Error(t, err)
	})

	t.Run("value of wrong type is returned as error", func(t *testing.T) {
		s := newMemStorage(map[string]any{"port": "not-a-number"})

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("storage-type"), config.WithStorage(s))
		assert.Nil(t, cfg)
		assert.Error(t, err)
	})

	t.Run("value is not part of the error message", func(t *testing.T) {
		s := newMemStorage(map[string]any{"port": "s3cr3t-value"})

		_, _, err := config.Load[*TestLoadConfig](config.WithName("storage-secret"), config.WithStorage(s))
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "s3cr3t-value")
		assert.Contains(t, err.Error(), "storage port: ")
	})
}

func TestLoad_StorageMock(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CONFIG", "")

	require.NoError(t, os.Chdir(t.TempDir()))

	var _ config.Storage = (*config.MockStorage)(nil)

	t.Run("reads the storage once and only uses All", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		s := config.NewMockStorage(ctrl)

		// Get and Set have no expectation: any call fails the test.
		s.EXPECT().All(gomock.Any()).Return(map[string]any{"host": "mock.host.com", "sub.name": "mock-sub"}, nil).Times(1)

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("mock-once"), config.WithStorage(s))
		require.NoError(t, err)

		assert.Equal(t, "mock.host.com", cfg.Host)
		assert.Equal(t, "mock-sub", cfg.Sub.Name)
		assert.Equal(t, 8080, cfg.Port) // not in storage: default
	})

	t.Run("reads the storage on every load", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		s := config.NewMockStorage(ctrl)

		hosts := []string{"first.host.com", "second.host.com"}
		calls := 0

		s.EXPECT().All(gomock.Any()).DoAndReturn(func(context.Context) (map[string]any, error) {
			h := hosts[calls]
			calls++

			return map[string]any{"host": h}, nil
		}).Times(2)

		for _, want := range hosts {
			cfg, _, err := config.Load[*TestLoadConfig](config.WithName("mock-twice"), config.WithStorage(s))
			require.NoError(t, err)
			assert.Equal(t, want, cfg.Host)
		}
	})

	t.Run("nil map is accepted", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		s := config.NewMockStorage(ctrl)

		s.EXPECT().All(gomock.Any()).Return(nil, nil)

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("mock-nil"), config.WithStorage(s))
		require.NoError(t, err)
		assert.Equal(t, "localhost", cfg.Host)
	})

	t.Run("error from All is returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		s := config.NewMockStorage(ctrl)

		boom := errors.New("storage unavailable")
		s.EXPECT().All(gomock.Any()).Return(nil, boom)

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("mock-err"), config.WithStorage(s))
		assert.Nil(t, cfg)
		assert.ErrorIs(t, err, boom)
	})

	t.Run("unknown key is returned as error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		s := config.NewMockStorage(ctrl)

		s.EXPECT().All(gomock.Any()).Return(map[string]any{"does.not.exist": "x"}, nil)

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("mock-invalid"), config.WithStorage(s))
		assert.Nil(t, cfg)
		assert.Error(t, err)
	})

	t.Run("WithContext passes the context to the storage", func(t *testing.T) {
		type key struct{}

		ctrl := gomock.NewController(t)
		s := config.NewMockStorage(ctrl)

		ctx := context.WithValue(context.Background(), key{}, "marker")

		s.EXPECT().All(ctx).Return(map[string]any{"host": "ctx.host.com"}, nil)

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("mock-ctx"), config.WithStorage(s), config.WithContext(ctx))
		require.NoError(t, err)
		assert.Equal(t, "ctx.host.com", cfg.Host)
	})

	t.Run("a background context is used by default", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		s := config.NewMockStorage(ctrl)

		s.EXPECT().All(gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context) (map[string]any, error) {
			assert.NoError(t, ctx.Err())

			return nil, nil
		})

		_, _, err := config.Load[*TestLoadConfig](config.WithName("mock-noctx"), config.WithStorage(s))
		require.NoError(t, err)
	})

	t.Run("a nil context falls back to a background context", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		s := config.NewMockStorage(ctrl)

		s.EXPECT().All(gomock.Not(gomock.Nil())).Return(nil, nil)

		_, _, err := config.Load[*TestLoadConfig](config.WithName("mock-nilctx"), config.WithStorage(s), config.WithContext(nil)) //nolint:staticcheck // nil is the case under test
		require.NoError(t, err)
	})

	t.Run("a canceled context is seen by the storage", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		s := config.NewMockStorage(ctrl)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		s.EXPECT().All(gomock.Any()).DoAndReturn(func(ctx context.Context) (map[string]any, error) {
			return nil, ctx.Err()
		})

		cfg, _, err := config.Load[*TestLoadConfig](config.WithName("mock-canceled"), config.WithStorage(s), config.WithContext(ctx))
		assert.Nil(t, cfg)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("the last WithStorage option wins", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		first := config.NewMockStorage(ctrl)
		second := config.NewMockStorage(ctrl)

		// first has no expectation: it must not be called.
		second.EXPECT().All(gomock.Any()).Return(map[string]any{"host": "second.host.com"}, nil)

		cfg, _, err := config.Load[*TestLoadConfig](
			config.WithName("mock-last"),
			config.WithStorage(first),
			config.WithStorage(second),
		)
		require.NoError(t, err)
		assert.Equal(t, "second.host.com", cfg.Host)
	})

	t.Run("storage is not read when the config file is invalid", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		s := config.NewMockStorage(ctrl) // no expectation: All must not be called

		file := filepath.Join(t.TempDir(), "broken.json")
		require.NoError(t, os.WriteFile(file, []byte(`{"host": "bad",`), 0644))

		_, _, err := config.Load[*TestLoadConfig](config.WithFile(file), config.WithStorage(s))
		assert.Error(t, err)
	})
}
