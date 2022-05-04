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
	"io/ioutil"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
)

func TestDetermineMissingIDsEmptyData(t *testing.T) {
	oldmails := []oldmail{}
	uids := []uid{}

	ranges, err := determineMissingIDs(oldmails, uids)

	assert.NoError(t, err)
	assert.Equal(t, []rangeT{}, ranges)
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

	ranges, err := determineMissingIDs(oldmails, uids)

	assert.NoError(t, err)
	assert.Equal(t, orgUIDs, uids)
	assert.Equal(t, []rangeT{}, ranges)
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

	ranges, err := determineMissingIDs(oldmails, uids)

	assert.NoError(t, err)
	assert.Equal(t, orgUIDs, uids)
	// This means that the emails that are located in the index interval [2, 4) in the "uids" slice
	// are not on disk. As common in maths, [ denotes a closed interval while ) denotes an open
	// interval. That is, missing data is `uids[2:4]`.
	assert.Equal(t, []rangeT{{start: 2, end: 4}}, ranges)
}

func TestDetermineMissingIDsMismatchesInRemoteData(t *testing.T) {
	uids := []uid{
		{Mbox: 1, Message: 1},
		{Mbox: 0, Message: 5},
		{Mbox: 0, Message: 6},
	}

	// UIDs are not consistent in uids slice.
	_, err := determineMissingIDs([]oldmail{}, uids)
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
	_, err := determineMissingIDs(oldmails, uids)
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

	ranges, err := determineMissingIDs(oldmails, uids)

	assert.NoError(t, err)
	assert.Equal(t, []rangeT{{start: 2, end: 3}, {start: 5, end: 6}}, ranges)
}

// This re-uses the mockEmail struct from the email tests.
func buildFakeEmail(hasNoData bool) *mockEmail {
	someTime := time.Now()
	msg := mockEmail{}
	if hasNoData {
		msg.On("Format").Return(
			// This email misses all fields, which causes errors.
			[]interface{}{},
		)
	} else {
		msg.On("Format").Return(
			// This email has all the fields we require.
			[]interface{}{
				imap.RawString("uid header"),
				uint32(1),
				imap.RawString("time header"),
				someTime,
				"rfc822 header",
				"actual content",
			},
		)
	}
	return &msg
}

func TestStreamingDeliverySuccessDespiteOneError(t *testing.T) {
	// Set up output directory and input channels.
	tmpdir := setUpEmptyMaildir(t, "folder", "oldmail")
	maildir := filepath.Join(tmpdir, "folder")
	resultDir := filepath.Join(maildir, "new")

	// Set up goroutine providing input and remember mocks to later assert on them.
	mocks := []*mockEmail{}
	msgChan := make(chan emailOps)
	go func() {
		for i := 0; i < 10; i++ {
			// Nine of these emails will provide all the data we need to store them. One, the one at
			// i==5, will not have any data. We use this to ensure we can continue processing emails
			// even though one has problems.
			msg := buildFakeEmail(i == 5)
			msgChan <- msg
			mocks = append(mocks, msg)
		}
		close(msgChan)
	}()

	var wg, stwg sync.WaitGroup
	// Delay actual operations until the entire pipeline has been set up.
	stwg.Add(1)

	uidvalidity := 0

	oldmailChan, errCountPtr := streamingDelivery(msgChan, maildir, uidvalidity, &wg, &stwg)
	assert.Zero(t, *errCountPtr)

	// Wait a while and check that nothing has happened yet.
	time.Sleep(time.Millisecond * 100) // nolint: gomnd
	content, err := ioutil.ReadDir(resultDir)
	assert.NoError(t, err)
	assert.Empty(t, content)
	assert.Empty(t, mocks)

	// Actually trigger operations and read from output channel.
	stwg.Done()
	oldmails := []oldmail{}
	for om := range oldmailChan {
		oldmails = append(oldmails, om)
	}
	wg.Wait()
	assert.Equal(t, 1, *errCountPtr)
	assert.Equal(t, 9, len(oldmails))

	// Ensure that we did write as many emails to the directory as we put in, apart from the one
	// that lacked data.
	content, err = ioutil.ReadDir(resultDir)
	assert.NoError(t, err)
	assert.Equal(t, 9, len(content))
	assert.Equal(t, 10, len(mocks))

	// Ensure that each email had its Format method called.
	for _, msg := range mocks {
		msg.AssertExpectations(t)
	}
}
