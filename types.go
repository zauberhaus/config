// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

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
