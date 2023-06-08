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
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendMailboxDisallowWriteOperations(t *testing.T) {
	mb := igrabMailbox{}

	err := mb.CopyMessages(true, nil, "")
	assert.ErrorIs(t, err, errReadOnlyServer)

	err = mb.CreateMessage(nil, time.Now(), nil)
	assert.ErrorIs(t, err, errReadOnlyServer)

	err = mb.Expunge()
	assert.ErrorIs(t, err, errReadOnlyServer)

	// Mailbox has not been changed.
	assert.Equal(t, igrabMailbox{}, mb)
}

func TestBackendMailboxNoops(t *testing.T) {
	mb := igrabMailbox{}

	err := mb.Check()
	assert.NoError(t, err)

	err = mb.SetSubscribed(true)
	assert.NoError(t, err)

	// Mailbox has not been changed.
	assert.Equal(t, igrabMailbox{}, mb)
}

func TestBackendMailboxNameInfoStatus(t *testing.T) {
	mb := igrabMailbox{
		maildir:  maildirPathT{base: "base", folder: "folder"},
		messages: []*igrabMessage{{msg: &memory.Message{Flags: []string{"\\Seen"}}}},
	}

	name := mb.Name()
	assert.Equal(t, "folder", name)

	info, err := mb.Info()
	assert.NoError(t, err)
	assert.Equal(t, "folder", info.Name)

	statusItems := []imap.StatusItem{
		imap.StatusMessages,
		imap.StatusUidNext,
		imap.StatusUidValidity,
		imap.StatusRecent,
		imap.StatusUnseen,
		imap.StatusAppendLimit,
	}
	status, err := mb.Status(statusItems)
	assert.NoError(t, err)

	assert.Equal(t, "folder", status.Name)
	assert.True(t, status.ReadOnly)
}

func TestBackendMailboxUpdateListSearch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "email")
	err := os.WriteFile(path, []byte(testBody), 0600)
	require.NoError(t, err)

	mb := igrabMailbox{
		maildir: maildirPathT{base: "base", folder: "folder"},
		messages: []*igrabMessage{
			{
				path:   path,
				filled: false,
				lock:   &sync.Mutex{},
				msg:    &memory.Message{Uid: 1},
			},
			{
				// The second message cannot be retrieved because there is an error reading in the
				// nonexisting file. Such messages are filtered out.
				path:   path + "i_do_not_exist",
				filled: false,
				lock:   &sync.Mutex{},
				msg:    &memory.Message{Uid: 2},
			},
		},
	}

	seqSet := &imap.SeqSet{}
	seqSet.AddNum(1)

	err = mb.UpdateMessagesFlags(true, seqSet, imap.SetFlags, []string{"\\Seen"})
	assert.NoError(t, err)

	seqSet.AddNum(2)

	receiver := make(chan *imap.Message, 2)

	err = mb.ListMessages(true, seqSet, []imap.FetchItem{imap.FetchBody}, receiver)
	assert.NoError(t, err)

	num := 0
	for range receiver {
		num++
	}
	assert.Equal(t, 1, num)

	uids, err := mb.SearchMessages(true, &imap.SearchCriteria{Uid: seqSet})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(uids))
}

func TestBackendMailboxAddMessagesDirMissingError(t *testing.T) {
	path := t.TempDir()
	mb := igrabMailbox{maildir: maildirPathT{base: path, folder: "folder"}}
	err := mb.addMessages()
	assert.Error(t, err)
}

func TestBackendMailboxAddmessages(t *testing.T) {
	tmpdir := filepath.Join(t.TempDir())
	for _, dir := range []string{"new", "cur"} {
		path := filepath.Join(tmpdir, "inbox", dir, "email_"+dir)
		err := os.MkdirAll(filepath.Dir(path), 0700)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(testBody), 0600)
		require.NoError(t, err)
	}

	mb := igrabMailbox{maildir: maildirPathT{base: tmpdir, folder: "inbox"}}

	err := mb.addMessages()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(mb.messages))

	for _, msg := range mb.messages {
		err := msg.fill()
		assert.NoError(t, err)
	}
}
