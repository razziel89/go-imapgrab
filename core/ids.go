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

import "fmt"

// Determine the UIDs of emails that have not yet been downloaded.
func determineMissingUIDs(oldmails []oldmail, uids []uid) ([]int, error) {
	// Check special cases such as an empty mailbox or uidvalidities that do not agree.
	if len(uids) == 0 {
		return []int{}, nil
	}
	uidvalidity := uids[0].Mbox
	for _, msg := range uids {
		if msg.Mbox != uidvalidity {
			err := fmt.Errorf("inconsistent UID validity on retrieved data")
			return []int{}, err
		}
	}
	for _, msg := range oldmails {
		if msg.uidValidity != uidvalidity {
			err := fmt.Errorf("inconsistent UID validity on stored data")
			return []int{}, err
		}
	}

	// Add the UIDs of the oldmail data (the data stored on disk) to a map to simplify determining
	// whether we've already downloaded some message.
	oldmailUIDs := make(map[int]struct{}, len(oldmails))
	for _, msg := range oldmails {
		oldmailUIDs[msg.uid] = struct{}{}
	}

	missingUIDs := []int{}
	// Determine which UIDs are missing on disk.
	for _, msg := range uids {
		if _, found := oldmailUIDs[msg.Message]; !found {
			missingUIDs = append(missingUIDs, msg.Message)
		}
	}

	return missingUIDs, nil
}
