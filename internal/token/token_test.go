// Copyright 2024 Matthew P. Dargan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package token

import "testing"

func TestNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		fields  []string
		row     []string
		wantErr bool
		want    string
	}{
		{
			name:    "empty fields",
			wantErr: true,
		},
		{
			name:    "fields and row must have the same length",
			fields:  []string{"a"},
			row:     []string{"b", "c"},
			wantErr: true,
		},
		{
			name:   "valid fields and row",
			fields: []string{"a", "b"},
			row:    []string{"1.0", "2.0"},
			want:   "\"a\": 1.0\n\"b\": 2.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ts, err := New(tt.fields, tt.row)
			t.Logf("ts: %v, err: %v", ts, err)
			if (err != nil) != tt.wantErr {
				t.Errorf("New(%v, %v) error = %v", tt.fields, tt.row, err)
			}
			if !tt.wantErr && ts != tt.want {
				t.Errorf("New(%v, %v) = %v, want %v", tt.fields, tt.row, ts, tt.want)
			}
		})
	}
}
