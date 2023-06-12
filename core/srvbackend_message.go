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
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/memory"
)

const (
	maxMemMegabytesEnv = "IGRAB_MAX_SERVER_CACHE_MB"
)

// We have a file-backed local storage. To be able to easily view the messages, the files will have
// to be read in. However, mailboxes can become very large, which would result in very high RAM
// usage if all the messages are being kept in memory for a long period of time. Thus, we use this
// implicit buffer that will
type backendMessageMemory struct {
	knownBytes int
	maxBytes   int
	messages   map[*serverMessage]bool
	lock       *sync.Mutex
}

var backendMem = backendMessageMemory{
	knownBytes: 0,
	// This means we will clear memory as soon as reading in a new message exceeds a storage of
	// about 100MB.
	maxBytes: intFromEnvWithDefault(maxMemMegabytesEnv, 100) * 1_000_000, //nolint:gomnd
	messages: map[*serverMessage]bool{},
	lock:     &sync.Mutex{},
}

func intFromEnvWithDefault(envVar string, defaultVal int) int {
	val, err := strconv.Atoi(os.Getenv(envVar))
	if err != nil {
		return defaultVal
	}
	return val
}

func clearBackendMessageMemoryIfNeeded(msg *serverMessage, newSize int) {
	backendMem.lock.Lock()
	defer backendMem.lock.Unlock()

	// Only consider clearing the memory if the message is not yet known.
	if _, found := backendMem.messages[msg]; !found {
		if newSize+backendMem.knownBytes > backendMem.maxBytes {
			for msg := range backendMem.messages {
				msg.clear()
				delete(backendMem.messages, msg)
			}
			backendMem.knownBytes = 0
		}
		backendMem.messages[msg] = true
		backendMem.knownBytes += newSize
	}
}

type serverMessage struct {
	path   string
	filled bool
	lock   *sync.Mutex

	msg *memory.Message
}

func (m *serverMessage) Fetch(seqNum uint32, items []imap.FetchItem) (*imap.Message, error) {
	err := m.fill()
	if err != nil {
		return nil, err
	}
	return m.msg.Fetch(seqNum, items)
}

func (m *serverMessage) Match(seqNum uint32, c *imap.SearchCriteria) (bool, error) {
	err := m.fill()
	if err != nil {
		return false, err
	}
	return m.msg.Match(seqNum, c)
}

func (m *serverMessage) fill() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.filled {
		return nil
	}
	// Fill only once if not yet filled.
	body, err := os.ReadFile(m.path)
	if err == nil {
		m.msg.Size = uint32(len(body))
		m.msg.Body = body
		m.filled = true
	}
	clearBackendMessageMemoryIfNeeded(m, len(body))
	logInfo(fmt.Sprintf("read %d bytes from %s", len(body), m.path))
	return err
}

func (m *serverMessage) clear() {
	m.lock.Lock()
	defer m.lock.Unlock()
	size := m.msg.Size
	m.msg.Size = 0
	m.msg.Body = nil
	m.filled = false
	logInfo(fmt.Sprintf("cleared %d bytes from %s", size, m.path))
}
