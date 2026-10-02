// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

//go:generate go run go.uber.org/mock/mockgen@latest --typed --write_package_comment=false -destination=storage_mock.go -source=$GOFILE -package=$GOPACKAGE

package config

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
	All() (map[string]any, error)
	Get(key string) (any, error)
	Set(key string, val any) error
}
