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
	"errors"
	"fmt"
	"io/fs"
	"os"
)

func isFile(path string) bool {
	stat, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	// We consider anything that exists and is no directory to be a file. This could be symlinks or
	// pipes or something similar. For the purpose of this tool, that distinction is likely not
	// relevant.
	return !stat.IsDir()
}

func isDir(path string) bool {
	stat, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return stat.IsDir()
}

func touch(path string, perm int) error {
	return os.WriteFile(path, []byte{}, filePerm)
}

type fileOps interface {
	Write(b []byte) (n int, err error)
	Close() error
	Read(p []byte) (n int, err error)
}

var openFile = openFileImpl

func openFileImpl(name string, flag int, perm fs.FileMode) (fileOps, error) {
	return os.OpenFile(name, flag, perm) // nolint: gosec
}

func errorIfExists(path, message string) error {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf(message)
}
