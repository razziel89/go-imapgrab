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
	"sort"

	"github.com/emersion/go-imap"
)

func determineMissingUIDs(oldmails []oldmail, uids []UID) (ranges []Range, err error) {
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

	// Sort the server-side information with respect to UIDs. That can be prohibitive for very large
	// mailboxes but it is the easiest way to find out which emails are present in both lists
	// without quadratic runtime (assuming that the sorting algorithm scales better than
	// quadratically). Most likely, the inputs will already be sorted in this manner.
	uidLess := func(i, j int) bool {
		return uids[i].Message < uids[j].Message
	}
	sort.Slice(uids, uidLess)

	// Add the UIDs of the oldmail data (the data stored on disk) to a map to simplify determining
	// whether we've already downloaded some message.
	oldmailUIDs := make(map[int]struct{}, len(oldmails))
	for _, msg := range oldmails {
		oldmailUIDs[msg.uid] = struct{}{}
	}

	// Determine which UIDs are missing on disk. The resulting structure will already be sorted.
	missingUIDs := []int{}
	for _, msg := range uids {
		if _, found := oldmailUIDs[msg.Message]; !found {
			missingUIDs = append(missingUIDs, msg.Message)
		}
	}
	if len(missingUIDs) == 0 {
		// All's well, everything is already on disk.
		return
	}

	// Extract consecutive ranges of UIDs from the missing UIDs, which speeds up downloading. That
	// way, we avoid retrieving messages one at a time.
	start := missingUIDs[0]
	last := start
	for _, mis := range missingUIDs {
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

	// Retrieve information about which emails are present on the remote system and check which ones
	// are missing when comparing against those on disk.
	uids, _, err := getAllMessageUUIDsAndTimestamps(mbox, imapClient)
	if err != nil {
		return err
	}
	missingUIDRanges, err := determineMissingUIDs(oldmails, uids)
	if err != nil {
		return err
	}

	// Download missing emails and store them on disk.
	for _, missingRange := range missingUIDRanges {
		msgs, err := getMessageRange(mbox, imapClient, missingRange)
		if err != nil {
			return err
		}
		// Deliver each email to the `tmp` directory and move them to the `new` directory.
		_ = msgs
	}

	// Write out information about newly retrieved emails.
	logInfo("writing oldmail file")
	if err := writeOldmail(oldmails, oldmailPath+".new"); err != nil {
		return err
	}
	logInfo("wrote new oldmail file")

	return nil
}
