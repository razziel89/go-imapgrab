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
	"path/filepath"
	"sync"
	"testing"

	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDownloader struct {
	messages      []*mockEmail
	messageChan   chan emailOps
	delivered     []oldmail
	deliveredChan chan oldmail
	t             *testing.T
	mock.Mock
}

func (m *mockDownloader) selectFolder(folder string) (*imap.MailboxStatus, error) {
	args := m.Called(folder)
	return args.Get(0).(*imap.MailboxStatus), args.Error(1)
}

func (m *mockDownloader) getAllMessageUUIDs(mbox *imap.MailboxStatus) ([]uid, error) {
	args := m.Called(mbox)
	return args.Get(0).([]uid), args.Error(1)
}

func (m *mockDownloader) streamingOldmailWriteout(
	deliveredChan <-chan oldmail, oldmailPath string, wg, startWg *sync.WaitGroup,
) (*int, error) {
	args := m.Called(deliveredChan, oldmailPath, wg, startWg)
	wg.Add(1)
	go func() {
		startWg.Wait()
		idx := 0
		for om := range deliveredChan {
			assert.Equal(m.t, om, m.delivered[idx])
			idx++
		}
		wg.Done()
	}()
	return args.Get(0).(*int), args.Error(1)
}

func (m *mockDownloader) streamingRetrieval(
	mbox *imap.MailboxStatus, missingIDRanges []rangeT, wg, startWg *sync.WaitGroup,
) (<-chan emailOps, *int, error) {
	args := m.Called(mbox, missingIDRanges, wg, startWg)
	wg.Add(1)
	go func() {
		startWg.Wait()
		for _, msg := range m.messages {
			var converted emailOps = msg
			m.messageChan <- converted
		}
		close(m.messageChan)
		wg.Done()
	}()
	return args.Get(0).(chan emailOps), args.Get(1).(*int), args.Error(2)
}

func (m *mockDownloader) streamingDelivery(
	messageChan <-chan emailOps, maildirPath string, uidvalidity int, wg, startWg *sync.WaitGroup,
) (<-chan oldmail, *int) {
	args := m.Called(messageChan, maildirPath, uidvalidity, wg, startWg)
	wg.Add(1)
	go func() {
		startWg.Wait()
		idx := 0
		for msg := range messageChan {
			assert.Equal(m.t, msg, m.messages[idx])
			m.deliveredChan <- m.delivered[idx]
			idx++
		}
		close(m.deliveredChan)
		wg.Done()
	}()
	return args.Get(0).(chan oldmail), args.Get(1).(*int)
}

func TestDownloadMissingEmailsToFolderSuccess(t *testing.T) {
	tmpdir := t.TempDir()
	maildirPath := maildirPathT{base: tmpdir, folder: "some-folder"}
	folderPath := maildirPath.folderPath()
	oldmailFileName := "some-oldmail-file"
	oldmailPath := filepath.Join(tmpdir, oldmailFileName)

	mbox := &imap.MailboxStatus{
		Name:        "some-folder",
		UidValidity: 42,
		Messages:    3,
	}
	uids := []uid{
		{Mbox: 42, Message: 1},
		{Mbox: 42, Message: 2},
		{Mbox: 42, Message: 3},
	}
	missingIDRanges := []rangeT{{start: 1, end: 4}}

	messages := []*mockEmail{
		{uid: 1}, {uid: 2}, {uid: 3},
	}
	messageChan := make(chan emailOps)
	var inMessageChan <-chan emailOps = messageChan
	var fetchErrCount int

	delivered := []oldmail{
		{uidValidity: 42, uid: 1},
		{uidValidity: 42, uid: 2},
		{uidValidity: 42, uid: 3},
	}
	deliveredChan := make(chan oldmail)
	var inDeliveredChan <-chan oldmail = deliveredChan
	var deliverErrCount int

	var oldmailErrCount int

	m := &mockDownloader{
		t:             t,
		messages:      messages,
		messageChan:   messageChan,
		delivered:     delivered,
		deliveredChan: deliveredChan,
	}

	m.On("selectFolder", "some-folder").Return(mbox, nil)
	m.On("getAllMessageUUIDs", mbox).Return(uids, nil)
	m.On("streamingRetrieval", mbox, missingIDRanges, mock.Anything, mock.Anything).
		Return(messageChan, &fetchErrCount, nil)
	m.On("streamingDelivery", inMessageChan, folderPath, 42, mock.Anything, mock.Anything).
		Return(deliveredChan, &deliverErrCount)
	m.On("streamingOldmailWriteout", inDeliveredChan, oldmailPath, mock.Anything, mock.Anything).
		Return(&oldmailErrCount, nil)

	err := downloadMissingEmailsToFolder(m, maildirPath, oldmailFileName)

	assert.NoError(t, err)
	m.AssertExpectations(t)
}
