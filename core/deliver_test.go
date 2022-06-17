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
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDeliverer struct {
	mock.Mock
}

func (m *mockDeliverer) deliverMessage(text string, maildirPath string) error {
	args := m.Called(text, maildirPath)
	return args.Error(0)
}

func (m *mockDeliverer) rfc822FromEmail(msg emailOps, uidvalidity int) (string, oldmail, error) {
	args := m.Called(msg, uidvalidity)
	return args.String(0), args.Get(1).(oldmail), args.Error(2)
}

func TestDelivererDeliverMessage(t *testing.T) {
	tmpdir := t.TempDir()
	missingDir := filepath.Join(tmpdir, "some", "dir", "that", "surely", "does", "not", "exist")

	deliverer := &deliverer{}
	err := deliverer.deliverMessage("some text", missingDir)

	assert.Error(t, err)
}

func TestDelivererRFC822FromEmail(t *testing.T) {
	msg := &mockEmail{uid: 42}
	msg.On("Format").Return([]interface{}{})

	deliverer := &deliverer{}
	_, _, err := deliverer.rfc822FromEmail(msg, 123)

	assert.Error(t, err)
	msg.AssertExpectations(t)
}

func TestStreamingDeliverySuccessDespiteOneError(t *testing.T) {
	m := &mockDeliverer{}

	mockEmails := []*mockEmail{}
	for i := 0; i < 10; i++ {
		msg := &mockEmail{uid: i}
		mockEmails = append(mockEmails, msg)
		om := oldmail{
			uidValidity: 42,
			uid:         i,
			timestamp:   12345,
		}
		var formatErr error
		if i == 5 {
			formatErr = fmt.Errorf("some error")
		}
		m.On("rfc822FromEmail", msg, 42).Return("actual content", om, formatErr)
		if formatErr == nil {
			m.On("deliverMessage", "actual content", "/some/path").Return(nil)
		}
	}

	// Set up goroutine providing input.
	msgChan := make(chan emailOps)
	go func() {
		for _, msg := range mockEmails {
			msgChan <- msg
		}
		close(msgChan)
	}()

	var wg, stwg sync.WaitGroup
	// Delay actual operations until the entire pipeline has been set up.
	stwg.Add(1)

	uidvalidity := 42
	maildir := "/some/path"

	oldmailChan, errCountPtr := streamingDelivery(m, msgChan, maildir, uidvalidity, &wg, &stwg)
	assert.Zero(t, *errCountPtr)

	// Wait a while and check that nothing has happened yet.
	time.Sleep(time.Millisecond * 100) // nolint: gomnd
	m.AssertNotCalled(t, "rfc822FromEmail", mock.Anything, mock.Anything)
	m.AssertNotCalled(t, "deliverMessage", mock.Anything, mock.Anything)

	// Actually trigger operations and read from output channel.
	stwg.Done()
	oldmails := []oldmail{}
	for om := range oldmailChan {
		oldmails = append(oldmails, om)
	}
	wg.Wait()

	m.AssertExpectations(t)
	assert.Equal(t, 1, *errCountPtr)
	assert.Equal(t, 9, len(oldmails))
}
