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
	"sync"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/memory"
)

type igrabMessage struct {
	path   string
	filled bool
	lock   *sync.Mutex

	msg *memory.Message
}

func (m *igrabMessage) Fetch(seqNum uint32, items []imap.FetchItem) (*imap.Message, error) {
	err := m.fill()
	if err != nil {
		return nil, err
	}
	return m.msg.Fetch(seqNum, items)
}

func (m *igrabMessage) Match(seqNum uint32, c *imap.SearchCriteria) (bool, error) {
	err := m.fill()
	if err != nil {
		return false, err
	}
	return m.msg.Match(seqNum, c)
}

func (m *igrabMessage) fill() error {
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
	}
	logInfo(fmt.Sprintf("read %d bytes from %s", len(body), m.path))
	return err
}
