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
	"sync"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

const (
	// Constants used to expand folder name specs.
	allSelector     = "_ALL_"
	gmailSelector   = "_Gmail_"
	removalSelector = "-"
	// Buffer for email delivery on disk.
	messageDeliveryBuffer = 10
)

// All gmail-specific folders, identified via prefixes..
const (
	gmailPrefix1 = "[Gmail]"
	gmailPrefix2 = "[Google Mail]"
)

func isGmailDir(dirName string) bool {
	return strings.HasPrefix(dirName, gmailPrefix1) || strings.HasPrefix(dirName, gmailPrefix2)
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

func streamingDelivery(
	messageChan <-chan *imap.Message, maildirPath string, uidvalidity int, wg, stwg *sync.WaitGroup,
) (returnedChan <-chan oldmail, errCountPtr *int) {
	var errCount int

	deliveredChan := make(chan oldmail, messageDeliveryBuffer)

	wg.Add(1)
	go func() {
		// Do not start before the entire pipeline has been set up.
		stwg.Wait()
		for msg := range messageChan {
			// Deliver each email to the `tmp` directory and move them to the `new` directory.
			text, oldmail, err := rfc822FromEmail(msg, uidvalidity)
			if err == nil {
				err = deliverMessage(text, maildirPath)
			}
			if err != nil {
				logError(err.Error())
				errCount++
				continue
			}
			deliveredChan <- oldmail
		}
		wg.Done()
		close(deliveredChan)
	}()

	return deliveredChan, &errCount
}

func downloadMissingEmailsToFolder(
	imapClient *client.Client, folder, maildirPath, oldmailName string,
) error {
	oldmails, oldmailPath, err := initMaildir(oldmailName, maildirPath)
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

	var wg, startWg sync.WaitGroup
	startWg.Add(1) // startWg is used to defer operations until the pipeline is set up.
	// Retrieve email information. This does not download the emails themselves yet.
	messageChan, fetchErrCount, err := streamingRetrieval(
		mbox, imapClient, missingIDRanges, &wg, &startWg,
	)
	if err != nil {
		return err
	}
	// Download missing emails and store them on disk.
	deliveredChan, deliverErrCount := streamingDelivery(
		messageChan, maildirPath, uidvalidity, &wg, &startWg,
	)
	if err != nil {
		return err
	}
	// Retrieve and write out information about all emails.
	oldmailErrCount, err := streamingOldmailWriteout(deliveredChan, oldmailPath, &wg, &startWg)
	if err != nil {
		return err
	}
	// Wait until all has been processed and report on errors.
	startWg.Done()
	wg.Wait()
	if *fetchErrCount > 0 || *deliverErrCount > 0 || *oldmailErrCount > 0 {
		return fmt.Errorf(
			"there were %d/%d/%d errors while: retrieving mail/delivering mail/writing to oldmail",
			*fetchErrCount, *deliverErrCount, *oldmailErrCount,
		)
	}

	return nil
}

// Perform fancy name replacements on folder names. For example, specifying _ALL_ causes all
// folders to be selected.
func expandFolders(folderSpecs, availableFolders []string) []string {
	logInfo(
		fmt.Sprintf("expanding folder spec '%s'", strings.Join(folderSpecs, logJoiner)),
	)
	logInfo(
		fmt.Sprintf("available folders are '%s'", strings.Join(availableFolders, logJoiner)),
	)

	// Convert to set to simplify manipulation.
	availableFoldersSet := setFromSlice(availableFolders)
	foldersSet := newOrderedSet(len(availableFolders))

	for _, folderSpec := range folderSpecs {
		if strings.HasPrefix(folderSpec, removalSelector) {
			folderSpec = strings.TrimPrefix(folderSpec, removalSelector)
			// Remove the specified directory.
			switch folderSpec {
			case allSelector:
				for _, removeMe := range availableFolders {
					foldersSet.remove(removeMe)
				}
			case gmailSelector:
				for _, removeMeCheck := range availableFolders {
					if isGmailDir(removeMeCheck) {
						foldersSet.remove(removeMeCheck)
					}
				}
			default:
				// Remove the specified folder, if it is known, log error otherwise.
				if !availableFoldersSet.has(folderSpec) {
					logError(fmt.Sprintf("ignoring attempted removal via spec %s", folderSpec))
				}
				foldersSet.remove(strings.TrimPrefix(folderSpec, removalSelector))
			}
		} else {
			// Add the specified directory.
			switch folderSpec {
			case allSelector:
				for _, addMe := range availableFolders {
					foldersSet.add(addMe)
				}
			case gmailSelector:
				for _, addMeCheck := range availableFolders {
					if isGmailDir(addMeCheck) {
						foldersSet.add(addMeCheck)
					}
				}
			default:
				foldersSet.add(folderSpec)
			}
		}
	}

	removed := foldersSet.keepUnion(availableFoldersSet)
	warning := fmt.Sprintf("unselecting nonexisting folders '%s'", strings.Join(removed, logJoiner))
	if len(removed) > 0 {
		logWarning(warning)
	}
	folders := foldersSet.orderedEntries()
	logInfo(fmt.Sprintf("expanded to folders '%s'", strings.Join(folders, logJoiner)))
	return folders
}
