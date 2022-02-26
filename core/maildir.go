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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	curMaildir = "cur"
	newMaildir = "new"
	tmpMaildir = "tmp"
	// The number of bits used for a random hex number to prevent name clashes.
	randomHexSize = 8
)

// A global delivery counter for this process used to determine a unique file name. A value of 0
// means no delivery has yet occurred.
var deliveryCount = 0

// Get a unique name for an email that will be delivered. Follow the process described here
// https://cr.yp.to/proto/maildir.html and implemented by getmail6 here
// https://github.com/getmail6/getmail6/blob/master/getmailcore/utilities.py#L274
//
// This function does not attempt to resolve conflicts. The chances for a naming conflict to occur
// are very, very small. For that to happen, two processes on two different machines that have the
// same hostname need to start a delivery at the very same time. Furthermore, they must have had
// delivered the exact same number of emails since launch and a 8-bit cryptographic random number
// must be identical. It is not clear how that should ever happen.
func newUniqueName() (string, error) {
	now := time.Now()
	timeInSecs := now.Unix()
	//nolint:gomnd
	microSecsOfTime := now.Nanosecond() / 1000 // Convert nano seconds to micro seconds.

	pid := os.Getpid()

	defer func() {
		// Increment the global delivery counter for this process. Increment even in an error case
		// since this counter is supposed to be unique for every message that this process has
		// processed.
		deliveryCount++
	}()

	// Extract an 8-bit random hex number.
	randomBytes := make([]byte, randomHexSize)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	randomHex := hex.EncodeToString(randomBytes)

	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	// Handle broken hostnames as per the above-linked description.
	hostname = strings.ReplaceAll(hostname, "/", "\\057")
	hostname = strings.ReplaceAll(hostname, ":", "\\072")

	filename := fmt.Sprintf(
		"%d.M%dP%dQ%dR%s.%s", timeInSecs, microSecsOfTime, pid, deliveryCount, randomHex, hostname,
	)

	// Sanity check against spaces in the file name.
	if strings.ContainsRune(filename, ' ') {
		return "", fmt.Errorf("whitespace detected in unique file name %s", filename)
	}

	return filename, nil
}

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

// Check whether a given path points to a maildir. This function checks for the existence of any
// required sub-directories and fails if they cannot be found. Furthermore, it checks for the
// existence of an oldmail file, parses it, and returns the information stored within it. It also
// returns the path to that oldmail file.
func initExistingMaildir(
	cfg IMAPConfig, maildirPath string,
) (oldmails []oldmail, oldmailFilePath string, err error) {
	if len(maildirPath) == 0 {
		err = fmt.Errorf("path to maildir cannot be empty")
		return
	}
	// Ensure the maildirPath has no trailing slashes and is generally as short as possible. This is
	// often called canonicalisation.
	maildirPath = filepath.Clean(maildirPath)

	logInfo(fmt.Sprintf("checking for sub-directories of possible maildir %s", maildirPath))
	if !isMaildir(cfg, maildirPath) {
		err = fmt.Errorf("given directory %s does not point to a maildir", maildirPath)
		return
	}
	logInfo("all sub-directories found")

	// Extract expected maildirPath of oldmail file.
	parent := filepath.Dir(maildirPath)
	base := filepath.Base(maildirPath)
	oldmailPath := filepath.Join(parent, oldmailFileName(cfg, base))

	logInfo(
		fmt.Sprintf("checking for and reading oldmail file of possible maildir %s", maildirPath),
	)
	oldmails, err = readOldmail(oldmailPath, maildirPath)
	if err != nil {
		return
	}
	logInfo("found and read oldmail file")

	return oldmails, oldmailPath, err
}

// ReadMaildir reads a maildir in and prints some information about it. This is usefiul for
// development and will probably not remain afterwards.
func ReadMaildir(cfg IMAPConfig, path string) error {
	oldmails, oldmailPath, err := initExistingMaildir(cfg, path)
	if err != nil {
		return err
	}

	logInfo("writing oldmail file")
	if err := writeOldmail(oldmails, oldmailPath+".new"); err != nil {
		return err
	}
	logInfo("wrote new oldmail file")

	for i := 0; i < 5; i++ {
		filename, err := newUniqueName()
		if err != nil {
			return err
		}
		logInfo(filename)
	}

	return nil
}
