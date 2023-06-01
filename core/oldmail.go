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
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	oldmailFields = 3
	oldmailFormat = "%d/%d_%d\n"
)

var (
	oldmailSepReplace = []byte("_")
	oldmailFormatSep  = []byte{0}
)

type oldmail struct {
	uidValidity int
	uid         int
	timestamp   int
}

// Provide a string representation for oldmail information.
func (om oldmail) String() string {
	timeStr := time.Unix(int64(om.timestamp), 0).UTC().String()
	return fmt.Sprintf("%d/%d -> %s", om.uidValidity, om.uid, timeStr)
}

func oldmailFileName(cfg IMAPConfig, folder string) string {
	return fmt.Sprintf("oldmail-%s-%d-%s-%s", cfg.Server, cfg.Port, cfg.User, folder)
}

// Read the oldmail information for a specific config. The oldmail config is found in the parent
// directory of a maildir. It might not be present. The oldmail file is called
// "oldmail-<SERVER_URL>-<PORT>-<USERNAME>-<INBOX>". It stores information about emails that have
// been processed during earlier runs and is used to determine which are new emails that need to be
// fetched.
//
// The format of each line of an oldmail file is <UIDVALIDITY>/<UID>\0<TIMESTAMP>. Here UIDVALIDITY
// is a unique identifier for a mailbox, UID is the unique identifier for an email within that
// mailbox, and TIMESTAMP is the unix timestamp when the message had been received by the server.
func readOldmail(oldmailPath string) (oldmails []oldmail, err error) {
	logInfo(fmt.Sprintf("reading oldmail file %s", oldmailPath))
	// Check for oldmail file.
	if !isFile(oldmailPath) {
		err = fmt.Errorf("oldmail file %s not found", oldmailPath)
		return
	}

	// Read the oldmail file in. This is required to determine which emails we have already
	// downloaded.
	handle, err := openFile(oldmailPath, os.O_RDONLY, filePerm) // nolint:gosec
	if err != nil {
		return
	}
	defer func() {
		if closeErr := handle.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	scanner := bufio.NewScanner(handle)
	for scanner.Scan() {
		// Read a line and parse it into an oldmail struct. The line has a null byte as separator.
		// Golang's strings don't seem to be able to contain those. Thus, read in the bytes, replace
		// the null byte by something else, and then parse the input. This is unnecessarily
		// complicated. Who had the bright idea of using a null byte as a separator, I wonder.
		line := string(bytes.ReplaceAll(scanner.Bytes(), oldmailFormatSep, oldmailSepReplace))

		// Parse the line.
		om := oldmail{}
		var scanned int
		scanned, err = fmt.Sscanf(line, oldmailFormat, &om.uidValidity, &om.uid, &om.timestamp)
		if scanned != oldmailFields {
			err = fmt.Errorf("too few fields in line %s", line)
		}
		if err != nil {
			return
		}

		oldmails = append(oldmails, om)
	}
	if err = scanner.Err(); err != nil {
		return
	}

	return oldmails, nil
}

// Write oldmail information to a path. See readOldmail for an explanation of the file format.
func streamingOldmailWriteout(
	oldmailChan <-chan oldmail, oldmailPath string, wg, stwg *sync.WaitGroup,
) (errCountPtr *int, err error) {
	logInfo(fmt.Sprintf("appending to oldmail file %s", oldmailPath))

	handle, err := openFile( // nolint:gosec
		oldmailPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm,
	)
	if err != nil {
		return nil, err
	}

	var errCount int
	wg.Add(1)
	go func() {
		var byteCount int
		var err error
		// Do not start before the entire pipeline has been set up.
		stwg.Wait()
		for om := range oldmailChan {
			// We do not expect errors to write out to fix themselves the next time around. Thus, we
			// only write if all writes had been successful so far. However, we read from
			// oldmailChan even if there has been an error. That way, we ensure that other
			// goroutines feeding the input channels do not hang.
			//
			// TODO: Find a way to signal the other goroutines that there has been an error. Until
			// now, we would continue to retrieve and parse emails but not write oldmail information
			// to disk.
			//
			// I don't expect many write-out errors in real life, though. Most failure
			// cases will be caught when opening the file above. Still, a fix would be nice.
			if err == nil {
				line := fmt.Sprintf(oldmailFormat, om.uidValidity, om.uid, om.timestamp)
				// Undo the replacement done when reading the file. See readOldmail for details.
				lineAsBytes := bytes.ReplaceAll([]byte(line), oldmailSepReplace, oldmailFormatSep)
				byteCount, err = handle.Write(lineAsBytes)
				if err != nil {
					logError(err.Error())
					errCount++
				} else {
					logInfo(fmt.Sprintf("wrote %d bytes to oldmail file", byteCount))
				}
			} else {
				logInfo("skipping oldmail writeout due to previous write error")
			}
		}

		err = handle.Close()
		if err != nil {
			logError(err.Error())
			errCount++
		}
		wg.Done()
	}()

	return &errCount, nil
}
