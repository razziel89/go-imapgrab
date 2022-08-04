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
	// Default permissions for the creation of new stuff.
	dirPerm  = 0700
	filePerm = 0600
)

// A global delivery counter for this process used to determine a unique file name. A value of 0
// means no delivery has yet occurred.
// TODO: use a mutex to enable multi-threaded downloads.
var deliveryCount = &threadSafeCounter{}

// Get a unique name for an email that will be delivered. Follow the process described here
// https://cr.yp.to/proto/maildir.html and implemented by getmail6 here
// https://github.com/getmail6/getmail6/blob/master/getmailcore/utilities.py#L274
//
// This function does not attempt to resolve conflicts. The chances for a naming conflict to occur
// are very, very small. For that to happen, two processes on two different machines that have the
// same hostname need to start a delivery at the very same time. Furthermore, they must have had
// delivered the exact same number of emails since launch and a 8-bit cryptographic random number
// must be identical. It is not clear how that should ever happen.
//
// You can inject a hostname, which allows to simulate generating a unique name for other
// environments. If none is injected (empty hostname), generate data for the current system.
func newUniqueName(hostname string) (filename string, err error) {
	localCount := deliveryCount.inc()

	// Determine data with functions that cannot fail.
	now := time.Now()
	timeInSecs := now.Unix()
	//nolint:gomnd
	microSecsOfTime := now.Nanosecond() / 1000 // Convert nano seconds to micro seconds.
	pid := os.Getpid()

	// Handle hostname.
	if hostname == "" {
		hostname, err = os.Hostname()
	}
	// Handle broken hostnames as per the above-linked description.
	hostname = strings.ReplaceAll(hostname, "/", "\\057")
	hostname = strings.ReplaceAll(hostname, ":", "\\072")

	// Handle the random hexadecimal string.
	randomBytes := make([]byte, randomHexSize)
	var randomHex string
	if err == nil {
		// Extract an 8-bit random hex number.
		_, err = rand.Read(randomBytes)
	}
	if err == nil {
		randomHex = hex.EncodeToString(randomBytes)
	}

	filename = fmt.Sprintf(
		"%d.M%dP%dQ%dR%s.%s", timeInSecs, microSecsOfTime, pid, localCount, randomHex, hostname,
	)

	// Sanity check against spaces in the file name.
	if err == nil && strings.ContainsRune(filename, ' ') {
		err = fmt.Errorf("whitespace detected in unique file name %s", filename)
	}

	return filename, err
}

// Function isMaildir checks whether a path is a path to a maildir. A maildir is a directory that
// contains the directories "cur", "new", and "tmp".
func isMaildir(path string) bool {
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
	oldmailName string, maildirPath maildirPathT,
) (oldmails []oldmail, oldmailFilePath string, err error) {
	logInfo("retrieving information about emails stored on disk")
	folderPath := maildirPath.folderPath()

	logInfo(fmt.Sprintf("checking for sub-directories of possible maildir %s", folderPath))
	if !isMaildir(folderPath) {
		err = fmt.Errorf("given directory %s does not point to a maildir", folderPath)
		return
	}
	logInfo("all sub-directories found")

	// Extract expected maildirPath of oldmail file.
	oldmailPath := filepath.Join(maildirPath.basePath(), oldmailName)

	logInfo(
		fmt.Sprintf("checking for and reading oldmail file of possible maildir %s", folderPath),
	)
	oldmails, err = readOldmail(oldmailPath)
	if err != nil {
		return
	}
	logInfo("found and read oldmail file")

	return oldmails, oldmailPath, err
}

// Initialize a maildir. If the given path already exists, only check whether the path is a maildir.
// If not, create the path first including all the required sub-directories and an empty oldmail
// file.
func initMaildir(oldmailName string, maildirPath maildirPathT) ([]oldmail, string, error) {
	logInfo(fmt.Sprintf("initializing maildir %s", maildirPath))
	basePath := maildirPath.basePath()
	folderPath := maildirPath.folderPath()
	// Replace each filesystem path separators by a dot. That way, we do not accidentally split
	// paths where we do not want to, which would cause us not to find the oldmail file or the
	// maildir.
	oldmailName = strings.ReplaceAll(oldmailName, string(os.PathSeparator), ".")
	if !isDir(folderPath) {
		logInfo(fmt.Sprintf("creating path to maildir %s and subdirectories", folderPath))
		err := os.MkdirAll(folderPath, dirPerm)
		for _, dir := range []string{newMaildir, curMaildir, tmpMaildir} {
			joined := filepath.Join(folderPath, dir)
			if err == nil {
				err = os.MkdirAll(joined, dirPerm)
			}
		}
		if err == nil {
			err = touch(filepath.Join(basePath, oldmailName), filePerm)
		}
		if err != nil {
			return []oldmail{}, "", err
		}
	}
	return initExistingMaildir(oldmailName, maildirPath)
}

// Write an email to the tmp sub-directory of a maildir with an appropriate, unique name and then
// move it to new sub-directory as mandated by the maildir specs.
func deliverMessage(rfc822 string, basePath string) (err error) {
	// Determine relevant paths.
	var fileName, tmpPath, newPath string
	fileName, err = newUniqueName("")
	if err == nil {
		tmpPath = filepath.Join(basePath, tmpMaildir, fileName)
		newPath = filepath.Join(basePath, newMaildir, fileName)
		err = errorIfExists(tmpPath, fmt.Sprintf("unique file name '%s' is not unique", tmpPath))
	}
	// Write rfc822 to file.
	if err == nil {
		logInfo(fmt.Sprintf("writing new email to file %s", tmpPath))
		err = os.WriteFile(tmpPath, []byte(rfc822), filePerm) //nolint:gosec
	}
	// Move to new location but only if there exist no other file at that location, yet. This is in
	// accordance with the maildir specs. It might take a while to write out a file, which means
	// there could be a race condition here.
	if err == nil {
		logInfo(fmt.Sprintf("moving email to permanent storage location %s", newPath))
		err = errorIfExists(newPath, fmt.Sprintf("permanent storage '%s' already exists", newPath))
	}
	if err == nil {
		err = os.Rename(tmpPath, newPath)
	}
	return err
}
