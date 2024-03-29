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
	uids := []uidExt{}

	missingIDs, err := determineMissingUIDs(oldmails, uids)

	assert.NoError(t, err)
	assert.Equal(t, []uid{}, missingIDs)
}

func TestDetermineMissingIDsEverythingDownloaded(t *testing.T) {
	oldmails := []oldmail{
		// Note that UIDs start at 1 according to the return values of IMAP servers.
		{uidFolder: 0, uid: 1, timestamp: 0},
		{uidFolder: 0, uid: 2, timestamp: 0},
		{uidFolder: 0, uid: 3, timestamp: 0},
	}
	uids := []uidExt{
		{folder: 0, msg: 1},
		{folder: 0, msg: 3}, // 3 and 2 swapped deliberately.
		{folder: 0, msg: 2},
	}
	orgUIDs := make([]uidExt, len(uids))
	_ = copy(orgUIDs, uids)

	missingIDs, err := determineMissingUIDs(oldmails, uids)

	assert.NoError(t, err)
	assert.Equal(t, orgUIDs, uids)
	assert.Equal(t, []uid{}, missingIDs)
}

func TestDetermineMissingIDsSomeMissing(t *testing.T) {
	oldmails := []oldmail{
		{uidFolder: 0, uid: 1, timestamp: 0},
		{uidFolder: 0, uid: 2, timestamp: 0},
		{uidFolder: 0, uid: 3, timestamp: 0},
		{uidFolder: 0, uid: 4, timestamp: 0},
	}
	uids := []uidExt{
		{folder: 0, msg: 1},
		{folder: 0, msg: 5},
		{folder: 0, msg: 6},
	}
	orgUIDs := make([]uidExt, len(uids))
	_ = copy(orgUIDs, uids)

	missingIDs, err := determineMissingUIDs(oldmails, uids)

	assert.NoError(t, err)
	assert.Equal(t, orgUIDs, uids)
	// This means that the emails that are located in the index interval [2, 4) in the "uids" slice
	// are not on disk. As common in maths, [ denotes a closed interval while ) denotes an open
	// interval. That is, missing data is `uids[2:4]`.
	assert.Equal(t, []uid{5, 6}, missingIDs)
}

func TestDetermineMissingIDsMismatchesInRemoteData(t *testing.T) {
	uids := []uidExt{
		{folder: 1, msg: 1},
		{folder: 0, msg: 5},
		{folder: 0, msg: 6},
	}

	// UIDs are not consistent in uids slice.
	_, err := determineMissingUIDs([]oldmail{}, uids)
	assert.Error(t, err)
}

func TestDetermineMissingIDsMismatches(t *testing.T) {
	oldmails := []oldmail{
		{uidFolder: 1, uid: 1, timestamp: 0},
	}
	uids := []uidExt{
		{folder: 0, msg: 1},
	}

	// UIDs are not consistent between uid and oldmails slices.
	_, err := determineMissingUIDs(oldmails, uids)
	assert.Error(t, err)
}

func TestDetermineMissingIDsSomeMissingNonconsecutiveRanges(t *testing.T) {
	oldmails := []oldmail{
		{uidFolder: 0, uid: 1, timestamp: 0},
		{uidFolder: 0, uid: 3, timestamp: 0},
		{uidFolder: 0, uid: 4, timestamp: 0},
		{uidFolder: 0, uid: 6, timestamp: 0},
	}
	uids := []uidExt{
		{folder: 0, msg: 1},
		{folder: 0, msg: 2},
		{folder: 0, msg: 3},
		{folder: 0, msg: 4},
		{folder: 0, msg: 5},
		{folder: 0, msg: 6},
	}

	missingIDs, err := determineMissingUIDs(oldmails, uids)

	assert.NoError(t, err)
	assert.Equal(t, []uid{2, 5}, missingIDs)
}
