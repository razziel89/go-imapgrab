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
)

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
