/* A re-implementation of the amazing imapgrap in plain Golang.
Copyright (C) 2022  Torsten Long

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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasePath(t *testing.T) {
	handler := maildirPathT{base: "basepath", folder: "folder name"}

	assert.Equal(t, "basepath", handler.basePath())
}

func TestFolderPath(t *testing.T) {
	handler := maildirPathT{base: "basepath", folder: "folder name"}

	assert.Equal(t, "basepath"+string(os.PathSeparator)+"folder name", handler.folderPath())
}

func TestFolderName(t *testing.T) {
	handler := maildirPathT{base: "basepath", folder: "folder name"}

	assert.Equal(t, "folder name", handler.folderName())
}
