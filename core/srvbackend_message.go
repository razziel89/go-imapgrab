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
	"strconv"
	"sync"
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
	maxBytes: intFromEnvWithDefault(maxMemMegabytesEnv, 100) * 1_000_000, //nolint:mnd
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

// Simplified serverMessage for v2 - server backend uses v1 for now
type serverMessage struct {
	path string
	lock *sync.Mutex
}

func (m *serverMessage) clear() {
	// Simplified for v2 migration
}
