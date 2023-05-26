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

import "path/filepath"

// Type maildirPathT provides routines to manipulate paths that are required to handle maildirs.
type maildirPathT struct {
	base   string
	folder string
}

func (p maildirPathT) basePath() string {
	return filepath.Clean(p.base)
}

func (p maildirPathT) folderPath() string {
	return filepath.Join(filepath.Clean(p.base), p.folder)
}

func (p maildirPathT) folderName() string {
	return p.folder
}
