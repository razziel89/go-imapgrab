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

const testBody = "" +
	"From: some@one.org\r\n" +
	"To: someone@else.org\r\n" +
	"Subject: Just a message\r\n" +
	"Date: Wed, 11 May 2016 14:31:59 +0000\r\n" +
	"Message-ID: <0000000@localhost/>\r\n" +
	"Content-Type: text/plain\r\n" +
	"\r\n" +
	"Stuff!"

func TestBackendMessage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "email")
	err := os.WriteFile(path, []byte(testBody), 0600)
	require.NoError(t, err)

	msg := serverMessage{
		path:   path,
		filled: false,
		lock:   &sync.Mutex{},
		msg: &memory.Message{
			Date:  time.Now(),
			Uid:   1,
			Flags: []string{"\\Seen"},
			Size:  0,
			Body:  nil,
		},
	}

	_, err = msg.Fetch(1, nil)
	assert.NoError(t, err)

	assert.Equal(t, msg.msg.Body, []byte(testBody))
	assert.Equal(t, msg.msg.Size, uint32(len(testBody)))
	assert.True(t, msg.filled)

	_, err = msg.Match(1, imap.NewSearchCriteria())
	assert.NoError(t, err)
}

func TestBackendMessageError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "email")

	msg := serverMessage{
		path:   path,
		filled: false,
		lock:   &sync.Mutex{},
		msg: &memory.Message{
			Date:  time.Now(),
			Uid:   1,
			Flags: []string{"\\Seen"},
			Size:  0,
			Body:  nil,
		},
	}

	_, err := msg.Fetch(1, nil)
	assert.Error(t, err)

	assert.Empty(t, msg.msg.Body)
	assert.Zero(t, msg.msg.Size)

	_, err = msg.Match(1, imap.NewSearchCriteria())
	assert.Error(t, err)
}

func TestAutoClearMemory(t *testing.T) {
	orgBackendMem := backendMem
	defer func() { backendMem = orgBackendMem }()

	backendMem = backendMessageMemory{
		knownBytes: 0,
		// This value means we clean up for every newly read message.
		maxBytes: 0,
		messages: map[*serverMessage]bool{},
		lock:     &sync.Mutex{},
	}

	path := filepath.Join(t.TempDir(), "email")
	err := os.WriteFile(path, []byte(testBody), 0600)
	require.NoError(t, err)

	msg1 := &serverMessage{
		path: path,
		lock: &sync.Mutex{},
		msg: &memory.Message{
			Date:  time.Now(),
			Uid:   1,
			Flags: []string{"\\Seen"},
		},
	}

	msg2 := &serverMessage{
		path: path,
		lock: &sync.Mutex{},
		msg: &memory.Message{
			Date:  time.Now(),
			Uid:   2,
			Flags: []string{"\\Seen"},
		},
	}

	err = msg1.fill()
	assert.NoError(t, err)

	_, found1 := backendMem.messages[msg1]
	_, found2 := backendMem.messages[msg2]

	assert.True(t, found1)
	assert.False(t, found2)

	// Reading in message 2 will cause message 1 to be purged from memory.
	err = msg2.fill()
	assert.NoError(t, err)

	assert.Equal(t, 1, len(backendMem.messages))

	_, found1 = backendMem.messages[msg1]
	_, found2 = backendMem.messages[msg2]

	assert.False(t, found1)
	assert.True(t, found2)
}

func TestIntFromEnvWithDefault(t *testing.T) {
	t.Setenv("ENV_VAR", "42")
	val := intFromEnvWithDefault("ENV_VAR", 21)
	assert.Equal(t, 42, val)

	t.Setenv("ENV_VAR", "no int")
	val = intFromEnvWithDefault("ENV_VAR", 21)
	assert.Equal(t, 21, val)
}
