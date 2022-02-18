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

// TODO:
// deliver email: An email will be delivered to the "tmp" sub-directory first with a unique file
// name obtained via newUniqueName.
//
// store email: Only once that email has successfully been written to disk will it be moved (not
// copied) to the "new" sub-directory with the exact same name. Use "os.Rename" to do so as the call
// is atomic enough not to cause problems.
//
// Remember the information needed to generate oldmail entries for the emails thus delivered (or
// generate that content directly). It might make most sense to append a line to the oldmail file
// for each email that has been delivered as it is being delivered. It would be easier to implement,
// though, to remember all that information and write all out at once in the very end.

// A global delivery counter for this process used to determine a unique file name. A value of -1
// means no delivery has yet occurred.
var deliveryCount = -1

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

	// Increment the global delivery counter for this process.
	deliveryCount++

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

	for i := 0; i < 5; i++ {
		filename, err := newUniqueName()
		if err != nil {
			return err
		}
		logInfo(filename)
	}

	return nil
}
