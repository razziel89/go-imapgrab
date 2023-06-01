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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetermineMissingIDsEmptyData(t *testing.T) {
	oldmails := []oldmail{}
	uids := []uid{}

	missingIDs, err := determineMissingUIDs(oldmails, uids)

	assert.NoError(t, err)
	assert.Equal(t, []int{}, missingIDs)
}

func TestDetermineMissingIDsEverythingDownloaded(t *testing.T) {
	// The fields uidValidity and Mbox mean the same thing. The fields uid and Message also mean the
	// same thing. The remote server identifies messages by their position in uids instead of their
	// actual unique identifiers. Thus, it is important that the uids slice is not rearranged in any
	// way.
	oldmails := []oldmail{
		// Note that UIDs start at 1 according to the return values of IMAP servers.
		{uidValidity: 0, uid: 1, timestamp: 0},
		{uidValidity: 0, uid: 2, timestamp: 0},
		{uidValidity: 0, uid: 3, timestamp: 0},
	}
	uids := []uid{
		{Mbox: 0, Message: 1},
		{Mbox: 0, Message: 3}, // 3 and 2 swapped deliberately.
		{Mbox: 0, Message: 2},
	}
	orgUIDs := make([]uid, len(uids))
	_ = copy(orgUIDs, uids)

	missingIDs, err := determineMissingUIDs(oldmails, uids)

	assert.NoError(t, err)
	assert.Equal(t, orgUIDs, uids)
	assert.Equal(t, []int{}, missingIDs)
}

func TestDetermineMissingIDsSomeMissing(t *testing.T) {
	oldmails := []oldmail{
		{uidValidity: 0, uid: 1, timestamp: 0},
		{uidValidity: 0, uid: 2, timestamp: 0},
		{uidValidity: 0, uid: 3, timestamp: 0},
		{uidValidity: 0, uid: 4, timestamp: 0},
	}
	uids := []uid{
		{Mbox: 0, Message: 1},
		{Mbox: 0, Message: 5},
		{Mbox: 0, Message: 6},
	}
	orgUIDs := make([]uid, len(uids))
	_ = copy(orgUIDs, uids)

	missingIDs, err := determineMissingUIDs(oldmails, uids)

	assert.NoError(t, err)
	assert.Equal(t, orgUIDs, uids)
	// This means that the emails that are located in the index interval [2, 4) in the "uids" slice
	// are not on disk. As common in maths, [ denotes a closed interval while ) denotes an open
	// interval. That is, missing data is `uids[2:4]`.
	assert.Equal(t, []int{5, 6}, missingIDs)
}

func TestDetermineMissingIDsMismatchesInRemoteData(t *testing.T) {
	uids := []uid{
		{Mbox: 1, Message: 1},
		{Mbox: 0, Message: 5},
		{Mbox: 0, Message: 6},
	}

	// UIDs are not consistent in uids slice.
	_, err := determineMissingUIDs([]oldmail{}, uids)
	assert.Error(t, err)
}

func TestDetermineMissingIDsMismatches(t *testing.T) {
	oldmails := []oldmail{
		{uidValidity: 1, uid: 1, timestamp: 0},
	}
	uids := []uid{
		{Mbox: 0, Message: 1},
	}

	// UIDs are not consistent between uid and oldmails slices.
	_, err := determineMissingUIDs(oldmails, uids)
	assert.Error(t, err)
}

func TestDetermineMissingIDsSomeMissingNonconsecutiveRanges(t *testing.T) {
	oldmails := []oldmail{
		{uidValidity: 0, uid: 1, timestamp: 0},
		{uidValidity: 0, uid: 3, timestamp: 0},
		{uidValidity: 0, uid: 4, timestamp: 0},
		{uidValidity: 0, uid: 6, timestamp: 0},
	}
	uids := []uid{
		{Mbox: 0, Message: 1},
		{Mbox: 0, Message: 2},
		{Mbox: 0, Message: 3},
		{Mbox: 0, Message: 4},
		{Mbox: 0, Message: 5},
		{Mbox: 0, Message: 6},
	}

	missingIDs, err := determineMissingUIDs(oldmails, uids)

	assert.NoError(t, err)
	assert.Equal(t, []int{2, 5}, missingIDs)
}
