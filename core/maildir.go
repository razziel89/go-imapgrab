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
	"os"
	"path/filepath"
)

const (
	curMaildir = "new"
	newMaildir = "new"
	tmpMaildir = "new"
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

// Function isMaildir checks whether a path is a path to a maildir. A maildir is a directory that
// contains the directories "cur", "new", and "tmp".
func isMaildir(cfg IMAPConfig, path string) bool {
	// Check for sub-directories.
	for _, dir := range []string{newMaildir, curMaildir, tmpMaildir} {
		fullPath := filepath.Join(path, dir)
		if !isDir(fullPath) {
			return false
		}
	}
	return true
}

// ReadMaildir reads a maildir in and prints some information about it. This is usefiul for
// development and will probably not remain afterwards.
func ReadMaildir(cfg IMAPConfig, path string) error {
	if len(path) == 0 {
		return fmt.Errorf("path to maildir cannot be empty")
	}
	// Ensure the path has no trailing slashes and is generally as short as possible. This is often
	// called canonicalisation.
	path = filepath.Clean(path)

	logInfo(fmt.Sprintf("checking for sub-directories of possible maildir %s", path))
	if !isMaildir(cfg, path) {
		return fmt.Errorf("given directory %s does not point to a maildir", path)
	}
	logInfo("all sub-directories found")

	// Extract expected path of oldmail file.
	parent := filepath.Dir(path)
	base := filepath.Base(path)
	oldmailPath := filepath.Join(parent, oldmailName(cfg, base))

	logInfo(fmt.Sprintf("checking for and reading oldmail file of possible maildir %s", path))
	oldmails, err := readOldmail(oldmailPath, path)
	if err != nil {
		return err
	}
	logInfo("found and read oldmail file")

	logInfo("writing oldmail file")
	if err := writeOldmail(oldmails, oldmailPath+".new"); err != nil {
		return err
	}
	logInfo("wrote new oldmail file")

	return nil
}
