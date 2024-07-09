// Copyright 2024 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package token provides facilities for tokenizing data.
package token

import (
	"errors"
	"fmt"
	"strings"
)

// New returns tokens for the given fields and row.
func New(fields, row []string) (string, error) {
	if len(fields) == 0 {
		return "", errors.New("empty fields")
	}
	if len(fields) != len(row) {
		return "", errors.New("fields and row must have the same length")
	}
	ts := make([]string, len(fields))
	for i, f := range fields {
		ts[i] = fmt.Sprintf("%q: %s", f, row[i])
	}
	return strings.Join(ts, "\n"), nil
}
