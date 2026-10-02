// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package errors keeps configuration values out of error messages, because
// values can be secrets that would otherwise end up in logs.
package errors

import (
	"fmt"
	"strconv"
	"strings"
)

// Redacted replaces a value in an error message.
const Redacted = "***"

// minRawLen is the shortest value that is redacted where it appears unquoted.
// Shorter values would mangle unrelated parts of the message, e.g. every "1".
const minRawLen = 4

// Error is an error about the value of Key from Source (e.g. "env", "flag" or
// "storage") with the value removed from its message. Err is the original
// error, which still contains the value; it is only reachable through
// errors.Unwrap, errors.Is and errors.As.
type Error struct {
	Source string
	Key    string
	Err    error

	msg string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s %s: %s", e.Source, e.Key, e.msg)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Wrap returns err unchanged if its message doesn't contain value, and
// otherwise an *Error with the value redacted.
func Wrap(source string, key string, value any, err error) error {
	if err == nil {
		return nil
	}

	v := fmt.Sprint(value)
	if v == "" {
		return err
	}

	msg := err.Error()
	redacted := strings.ReplaceAll(msg, strconv.Quote(v), strconv.Quote(Redacted))

	if len(v) >= minRawLen {
		redacted = strings.ReplaceAll(redacted, v, Redacted)
	}

	if redacted == msg {
		return err
	}

	return &Error{Source: source, Key: key, Err: err, msg: redacted}
}
