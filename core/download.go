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
	"strings"

	"github.com/emersion/go-imap/client"
)

const (
	allSelector     = "_ALL_"
	gmailSelector   = "_Gmail_"
	removalSelector = "-"
)

// All gmail-specific folders.
var gmailList = []string{
	"[Gmail]",
	"[Gmail]/All Mail",
	"[Gmail]/Drafts",
	"[Gmail]/Sent Mail",
	"[Gmail]/Spam",
	"[Gmail]/Starred",
	"[Gmail]/Trash",
	"[Google Mail]",
	"[Google Mail]/All Mail",
	"[Google Mail]/Drafts",
	"[Google Mail]/Sent Mail",
	"[Google Mail]/Spam",
	"[Google Mail]/Starred",
	"[Google Mail]/Trash",
}

// Determine the indices of emails that have not yet been downloaded. The download process
// indentifies emails by their indices and not by their UIDs. Thus, we need to take the server-side
// information as is and not sort it in any way.
func determineMissingIDs(oldmails []oldmail, uids []uid) (ranges []rangeT, err error) {
	// Check special cases such as an empty mailbox or uidvalidities that do not agree.
	if len(uids) == 0 {
		return []rangeT{}, nil
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
			ranges = append(ranges, rangeT{start: start, end: last + 1})
			start = mis
		}
		last = mis
	}
	ranges = append(ranges, rangeT{start: start, end: last + 1})

	return ranges, nil
}

func downloadMissingEmailsToFolder(
	imapClient *client.Client, folder, maildirPath, oldmailName string,
) error {
	oldmails, oldmailPath, err := initExistingMaildir(oldmailName, maildirPath)
	if err != nil {
		return err
	}

	mbox, err := selectFolder(imapClient, folder)
	if err != nil {
		return err
	}
	uidvalidity := int(mbox.UidValidity)

	// Retrieve information about which emails are present on the remote system and check which ones
	// are missing when comparing against those on disk.
	uids, err := getAllMessageUUIDs(mbox, imapClient)
	if err != nil {
		return err
	}

	missingIDRanges, err := determineMissingIDs(oldmails, uids)
	if err != nil {
		return err
	}
	total := accumulateRanges(missingIDRanges)
	if total == 0 {
		logInfo("no new emails, nothing to be done")
		return nil
	}
	logInfo(fmt.Sprintf("will download %d new emails", total))

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
			err = deliverMessage(text, maildirPath)
			if err != nil {
				return err
			}
			oldmails = append(oldmails, oldmail)
		}
	}

	// Write out information about newly retrieved emails.
	if err := writeOldmail(oldmails, oldmailPath); err != nil {
		return err
	}

	return nil
}

// Perform fancy name replacements on folder names. For example, specifying _ALL_ causes all
// folders to be selected.
func expandFolders(folderSpecs, availableFolders []string) []string {
	logInfo(
		fmt.Sprintf("expanding folder spec '%s'", strings.Join(folderSpecs, logSliceJoiner)),
	)
	logInfo(
		fmt.Sprintf("available folders are '%s'", strings.Join(availableFolders, logSliceJoiner)),
	)

	// Convert to set to simplify manipulation.
	availableFoldersSet := setFromSlice(availableFolders)
	foldersSet := newOrderedSet(len(availableFolders))

	for _, folderSpec := range folderSpecs {
		if strings.HasPrefix(folderSpec, removalSelector) {
			// Remove the specified directory.
			switch folderSpec {
			case allSelector:
				// Remove all available directories, if any have been added yet.
				for _, removeMe := range availableFolders {
					foldersSet.remove(removeMe)
				}
			case gmailSelector:
				// Remove only the gmail-specific stuff.
				for _, removeMe := range gmailList {
					foldersSet.remove(removeMe)
				}
			default:
				// Remove the specified folder, if it is known, log error otherwise.
				if !availableFoldersSet.has(strings.TrimPrefix(folderSpec, removalSelector)) {
					logError(fmt.Sprintf("ignoring attempted removal via spec %s", folderSpec))
				}
				foldersSet.remove(strings.TrimPrefix(folderSpec, removalSelector))
			}
		} else {
			// Add the specified directory.
			switch folderSpec {
			case allSelector:
				// Add all available directories.
				for _, addMe := range availableFolders {
					foldersSet.add(addMe)
				}
			case gmailSelector:
				// Add only the gmail-specific stuff.
				for _, addMe := range gmailList {
					foldersSet.add(addMe)
				}
			default:
				foldersSet.add(folderSpec)
			}
		}
	}

	removed := foldersSet.keepUnion(availableFoldersSet)
	logWarning(
		fmt.Sprintf("unselecting non-existing folders '%s'", strings.Join(removed, logSliceJoiner)),
	)

	folders := foldersSet.orderedEntries()
	logInfo(fmt.Sprintf("expanded to folders '%s'", strings.Join(folders, logSliceJoiner)))
	return folders
}
