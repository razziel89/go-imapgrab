/* A re-implementation of the amazing imapgrap in plain Golang.
Copyright (C) 2022  Torsten Sachse

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOldmailToString(t *testing.T) {
	om := oldmail{
		uidValidity: 100,
		uid:         42,
		timestamp:   12345,
	}
	// Unix 12345 is equal to: Thu  1 Jan 03:25:45 UTC 1970
	expectedTimeFormat := "100/42 -> 1970-01-01 03:25:45 +0000 UTC"

	assert.Equal(t, expectedTimeFormat, om.String())
}

func TestOldmailFileName(t *testing.T) {
	name := oldmailFileName(
		IMAPConfig{
			Server:   "some_server",
			Port:     42,
			User:     "some_user",
			Password: "not contained in file name",
		},
		"some_folder",
	)

	assert.Equal(t, "oldmail-some_server-42-some_user-some_folder", name)
}

func TestOldmailRead(t *testing.T) {
	expectedOldmails := []oldmail{
		{uidValidity: 123, uid: 21, timestamp: 747},
		{uidValidity: 123, uid: 42, timestamp: 447},
		{uidValidity: 123, uid: 11, timestamp: 321},
		{uidValidity: 123, uid: 15, timestamp: 898},
		{uidValidity: 123, uid: 17, timestamp: 242},
	}
	oldmailContent := []byte(`123/21_747
123/42_447
123/11_321
123/15_898
123/17_242
`)
	// Replace "_" by the null character to work around go strings ignoring the null byte.
	oldmailContent = bytes.ReplaceAll(oldmailContent, []byte("_"), []byte{0})

	tmp := t.TempDir()
	tmpFile := filepath.Join(tmp, "tmpfile")

	err := os.WriteFile(tmpFile, oldmailContent, 0444)
	assert.NoError(t, err)

	oldmails, err := readOldmail(tmpFile)
	assert.NoError(t, err)

	assert.Equal(t, expectedOldmails, oldmails)
}
