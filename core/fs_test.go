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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsFile(t *testing.T) {
	tmp := t.TempDir()
	tmpFile := filepath.Join(tmp, "file")
	missingFile := filepath.Join(tmp, "missing_file")

	err := os.WriteFile(tmpFile, []byte{}, 0444)
	assert.NoError(t, err)

	assert.True(t, isFile(tmpFile))
	assert.False(t, isFile(tmp))
	assert.False(t, isFile(missingFile))
}

func TestIsDir(t *testing.T) {
	tmp := t.TempDir()
	tmpFile := filepath.Join(tmp, "file")
	missingDir := filepath.Join(tmp, "missing_dir")

	err := os.WriteFile(tmpFile, []byte{}, 0444)
	assert.NoError(t, err)

	assert.False(t, isDir(tmpFile))
	assert.True(t, isDir(tmp))
	assert.False(t, isDir(missingDir))
}

func TestTouch(t *testing.T) {
	tmp := t.TempDir()
	tmpFile := filepath.Join(tmp, "file")

	err := touch(tmpFile, 0444)
	assert.NoError(t, err)
}

func TestOpenFile(t *testing.T) {
	tmp := t.TempDir()
	tmpFile := filepath.Join(tmp, "file")

	_, err := openFile(tmpFile, os.O_CREATE|os.O_WRONLY, 0666)
	assert.NoError(t, err)
}
