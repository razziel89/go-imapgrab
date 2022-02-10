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

func oldmailName(cfg IMAPConfig, folder string) string {
	return fmt.Sprintf("oldmail-%s-%d-%s-%s", cfg.Server, cfg.Port, cfg.User, folder)
}

// Function isMaildir checks whether a path is a path to a maildir. A maildir is a directory that
// contains the directories "cur", "new", and "tmp". It also needs to have an oldmail file in its
// parent folder. See readOldmail for a description of what that file looks like and its content.
func isMaildir(cfg IMAPConfig, path string) bool {
	// Check for oldmail file.
	logInfo(fmt.Sprintf("checking for sub-directories of possible maildir %s", path))
	for _, dir := range []string{newMaildir, curMaildir, tmpMaildir} {
		fullPath := filepath.Join(path, dir)
		if !isDir(fullPath) {
			logInfo(fmt.Sprintf("cannot find required directory %s", fullPath))
			return false
		}
	}
	logInfo(fmt.Sprintf("checking for oldmail file of possible maildir %s", path))

	// Check for oldmail file.
	parent := filepath.Dir(path)
	base := filepath.Base(path)
	oldmail := filepath.Join(parent, oldmailName(cfg, base))
	logInfo(fmt.Sprintf("expected oldmail file is %s", oldmail))

	return isFile(oldmail)
}

// Read the oldmail information for a specific config. The oldmail config is found in the parent
// directory of a maildir. It might not be present. The oldmail file is called
// "oldmail-<SERVER_URL>-<PORT>-<USERNAME>-<INBOX>". It stores information about emails that have
// been processed during earlier runs and is used to determine which are new emails that need to be
// fetched.
func readOldmail(cfg IMAPConfig, path string) error {
	return nil
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
	if !isMaildir(cfg, path) {
		return fmt.Errorf("given directory %s does not point to a maildir", path)
	}
	err := readOldmail(cfg, path)
	if err != nil {
		return err
	}

	return nil
}
