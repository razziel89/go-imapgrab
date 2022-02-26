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
	"fmt"
	"os"
	"path/filepath"

	"github.com/emersion/go-imap"
)

// Determine the indices of emails that have not yet been downloaded. The download process
// indentifies emails by their indices and not by their UIDs. Thus, we need to take the server-side
// information as is and not sort it in any way.
func determineMissingIDs(oldmails []oldmail, uids []UID) (ranges []Range, err error) {
	// Check special cases such as an empty mailbox or uidvalidities that do not agree.
	if len(uids) == 0 {
		return []Range{}, nil
	}
	uidvalidity := uids[0].Mbox
	for _, msg := range uids {
		if msg.Mbox != uidvalidity {
			err = fmt.Errorf("inconsistent UID validity on retrieved data")
			return
		}
	}
	for _, msg := range oldmails {
		if msg.uidValidity != uidvalidity {
			err = fmt.Errorf("inconsistent UID validity on stored data")
			return
		}
	}

	// Add the UIDs of the oldmail data (the data stored on disk) to a map to simplify determining
	// whether we've already downloaded some message.
	oldmailUIDs := make(map[int]struct{}, len(oldmails))
	for _, msg := range oldmails {
		oldmailUIDs[msg.uid] = struct{}{}
	}

	// Determine which UIDs are missing on disk. The resulting structure will already be sorted.
	missingIDs := []int{}
	for msgIdx, msg := range uids {
		if _, found := oldmailUIDs[msg.Message]; !found {
			missingIDs = append(missingIDs, msgIdx+1) // Emails are identified starting at 1.
		}
	}
	if len(missingIDs) == 0 {
		// All's well, everything is already on disk.
		return
	}

	// Extract consecutive ranges of UIDs from the missing UIDs, which speeds up downloading. That
	// way, we avoid retrieving messages one at a time.
	start := missingIDs[0]
	last := start
	for _, mis := range missingIDs {
		if mis-last > 1 {
			ranges = append(ranges, Range{Start: start, End: last + 1})
			start = mis
		}
		last = mis
	}
	ranges = append(ranges, Range{Start: start, End: last + 1})

	return ranges, nil
}

func rfc822FromEmail(msg *imap.Message, uidvalidity int) (string, oldmail, error) {
	fields := msg.Format()
	if len(fields) != rfc822ExpectedNumFields {
		return "", oldmail{}, fmt.Errorf("cannot extract required RFC822 fields from email")
	}

	email := Email{}
	for _, field := range fields {
		if err := email.set(field); err != nil {
			return "", oldmail{}, fmt.Errorf("cannot extract email data: %s", err.Error())
		}
	}
	if !email.validate() {
		return "", oldmail{}, fmt.Errorf("cannot extract full email from reply")
	}

	text := email.String()
	oldmailInfo := oldmail{
		uid:         email.UID,
		uidValidity: uidvalidity,
		timestamp:   int(email.Timestamp.Unix()),
	}

	return text, oldmailInfo, nil
}

// DownloadFolder downloads all not yet downloaded email from a folder in a mailbox to a maildir.
// The oldmail file in the parent directory of the maildir is used to determine which emails have
// already been downloaded. According to the [maildir specs](https://cr.yp.to/proto/maildir.html),
// the email is first downloaded into the `tmp` sub-directory and then moved atomically to the `new`
// sub-directory.
///nolint:funlen
func DownloadFolder(cfg IMAPConfig, folder, maildirPath string) error {
	// Retrieve information about emails that have already been downloaded.
	oldmails, oldmailPath, err := initExistingMaildir(cfg, maildirPath)
	if err != nil {
		return err
	}

	// Authenticate against the remote server.
	imapClient, err := authenticateClient(cfg)
	if err != nil {
		return err
	}
	// Make sure to log out in the end if we logged in successfully.
	defer func() {
		// Don't overwrite the error if it has already been set.
		if logoutErr := imapClient.Logout(); logoutErr != nil && err == nil {
			err = logoutErr
		}
	}()
	mbox, err := selectFolder(imapClient, folder)
	if err != nil {
		return err
	}
	uidvalidity := int(mbox.UidValidity)
	_ = uidvalidity

	// Retrieve information about which emails are present on the remote system and check which ones
	// are missing when comparing against those on disk.
	uids, _, err := getAllMessageUUIDsAndTimestamps(mbox, imapClient)
	if err != nil {
		return err
	}
	logInfo(fmt.Sprintf("received information for %d emails", len(uids)))
	fmt.Println(uids)
	missingIDRanges, err := determineMissingIDs(oldmails, uids)
	if err != nil {
		return err
	}
	logInfo(fmt.Sprintf("will download %d new emails", len(missingIDRanges)))
	fmt.Println(missingIDRanges)

	// Download missing emails and store them on disk.
	for _, missingRange := range missingIDRanges {
		msgs, err := getMessageRange(mbox, imapClient, missingRange)
		if err != nil {
			return err
		}
		// Deliver each email to the `tmp` directory and move them to the `new` directory.
		for _, msg := range msgs {
			text, oldmail, err := rfc822FromEmail(msg, uidvalidity)
			if err != nil {
				return err
			}
			logInfo(fmt.Sprintf("downloaded email %s", oldmail))
			fileName, err := newUniqueName()
			if err != nil {
				return err
			}
			tmpPath := filepath.Join(maildirPath, tmpMaildir, fileName)
			newPath := filepath.Join(maildirPath, newMaildir, fileName)
			if isFile(tmpPath) {
				return fmt.Errorf("unique file name '%s' is not unique", tmpPath)
			}
			logInfo(fmt.Sprintf("writing new email to file %s", tmpPath))
			err = os.WriteFile(tmpPath, []byte(text), 0644) //nolint:gosec,gomnd
			if err != nil {
				return err
			}
			oldmails = append(oldmails, oldmail)
			logInfo(fmt.Sprintf("moving email to permanent storage location %s", newPath))
			if isFile(newPath) {
				return fmt.Errorf("permanent storate '%s' already exists", newPath)
			}
			err = os.Rename(tmpPath, newPath)
			if err != nil {
				return err
			}
		}
	}

	// Write out information about newly retrieved emails.
	logInfo(fmt.Sprintf("writing oldmail file %s", oldmailPath))
	if err := writeOldmail(oldmails, oldmailPath); err != nil {
		return err
	}
	logInfo("wrote new oldmail file")

	return nil
}
