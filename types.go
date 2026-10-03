// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

//go:generate go run go.uber.org/mock/mockgen@latest --typed --write_package_comment=false -destination=storage_mock.go -source=$GOFILE -package=$GOPACKAGE

package config

import "context"

type FileType int
type Extension struct {
	Name     string
	FileType FileType
}

const (
	UnknownFileType FileType = iota
	JSON
	YAML
)

type Storage interface {
	All(ctx context.Context) (map[string]any, error)
	Get(ctx context.Context, key string) (any, error)
	Set(ctx context.Context, key string, val any) error
	Delete(ctx context.Context, key string) error
}
