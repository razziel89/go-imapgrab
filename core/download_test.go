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
	missingUIDs []int, wg, startWg *sync.WaitGroup, in func() bool,
) (<-chan emailOps, *int, error) {
	args := m.Called(missingUIDs, wg, startWg, in)
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
		{Mbox: 42, Message: 1}, {Mbox: 42, Message: 2}, {Mbox: 42, Message: 3},
	}
	missingUIDs := []int{1, 2, 3}

	messages := []*mockEmail{
		{uid: 1}, {uid: 2}, {uid: 3},
	}
	messageChan := make(chan emailOps)
	var inMessageChan <-chan emailOps = messageChan
	var fetchErrCount int

	delivered := []oldmail{
		{uidValidity: 42, uid: 1}, {uidValidity: 42, uid: 2}, {uidValidity: 42, uid: 3},
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

	mi := &mockInterrupter{}
	mi.On("interrupted").Return(false)

	m.On("selectFolder", "some-folder").Return(mbox, nil)
	m.On("getAllMessageUUIDs", mbox).Return(uids, nil)
	m.On("streamingRetrieval",
		missingUIDs, mock.Anything, mock.Anything,
		// We cannot use functions in expectations. Thus use this construct instead.
		mock.AnythingOfType("func() bool"),
	).Return(messageChan, &fetchErrCount, nil)
	m.On("streamingDelivery", inMessageChan, folderPath, 42, mock.Anything, mock.Anything).
		Return(deliveredChan, &deliverErrCount)
	m.On("streamingOldmailWriteout", inDeliveredChan, oldmailPath, mock.Anything, mock.Anything).
		Return(&oldmailErrCount, nil)

	err := downloadMissingEmailsToFolder(m, maildirPath, oldmailFileName, mi)

	assert.NoError(t, err)
	m.AssertExpectations(t)
	mi.AssertExpectations(t)
}

func TestDownloadMissingEmailsToFolderPreparationError(t *testing.T) {
	tmpdir := t.TempDir()
	maildirPath := maildirPathT{base: tmpdir, folder: "some-folder"}
	oldmailFileName := "some-file"

	mbox := &imap.MailboxStatus{
		Name:        "some-folder",
		UidValidity: 42,
		Messages:    3,
	}

	m := &mockDownloader{t: t}

	m.On("selectFolder", "some-folder").Return(mbox, nil)

	mi := &mockInterrupter{}
	mi.On("interrupted").Return(true) // Simulate an interrupt.

	err := downloadMissingEmailsToFolder(m, maildirPath, oldmailFileName, mi)

	assert.Error(t, err)
	assert.Equal(t, "aborting due to user interrupt", err.Error())
	m.AssertExpectations(t)
}

func TestDownloadMissingEmailsToFolderPreparationNoNewEmails(t *testing.T) {
	tmpdir := t.TempDir()
	maildirPath := maildirPathT{base: tmpdir, folder: "some-folder"}
	oldmailFileName := "some-file"

	mbox := &imap.MailboxStatus{
		Name:        "some-folder",
		UidValidity: 42,
		Messages:    3,
	}
	// No emails so nothing will be downloaded.
	uids := []uid{}

	m := &mockDownloader{t: t}

	m.On("selectFolder", "some-folder").Return(mbox, nil)
	m.On("getAllMessageUUIDs", mbox).Return(uids, nil)

	mi := &mockInterrupter{}
	mi.On("interrupted").Return(false)

	err := downloadMissingEmailsToFolder(m, maildirPath, oldmailFileName, mi)

	assert.NoError(t, err)
	m.AssertExpectations(t)
}

func TestDownloadMissingEmailsToFolderDownloadError(t *testing.T) {
	// This test is almost identical to the success case. The only difference is that we increase
	// the error counters to test that such errors are reported in the very end.
	tmpdir := t.TempDir()
	maildirPath := maildirPathT{base: tmpdir, folder: "some-folder"}
	folderPath := maildirPath.folderPath()
	oldmailFileName := "some-oldmail-file"
	oldmailPath := filepath.Join(tmpdir, oldmailFileName)

	mbox := &imap.MailboxStatus{
		Name: "some-folder", UidValidity: 42, Messages: 3,
	}
	uids := []uid{
		{Mbox: 42, Message: 1}, {Mbox: 42, Message: 2}, {Mbox: 42, Message: 3},
	}
	missingUIDs := []int{1, 2, 3}

	messages := []*mockEmail{{uid: 1}, {uid: 2}, {uid: 3}}
	messageChan := make(chan emailOps)
	var inMessageChan <-chan emailOps = messageChan
	fetchErrCount := 1

	delivered := []oldmail{
		{uidValidity: 42, uid: 1}, {uidValidity: 42, uid: 2}, {uidValidity: 42, uid: 3},
	}
	deliveredChan := make(chan oldmail)
	var inDeliveredChan <-chan oldmail = deliveredChan
	deliverErrCount := 1
	oldmailErrCount := 1

	m := &mockDownloader{
		t:             t,
		messages:      messages,
		messageChan:   messageChan,
		delivered:     delivered,
		deliveredChan: deliveredChan,
	}

	mi := &mockInterrupter{}
	mi.On("interrupted").Return(false)

	m.On("selectFolder", "some-folder").Return(mbox, nil)
	m.On("getAllMessageUUIDs", mbox).Return(uids, nil)
	m.On("streamingRetrieval", missingUIDs, mock.Anything, mock.Anything,
		// We cannot use functions in expectations. Thus use this construct instead.
		mock.AnythingOfType("func() bool"),
	).Return(messageChan, &fetchErrCount, nil)
	m.On("streamingDelivery", inMessageChan, folderPath, 42, mock.Anything, mock.Anything).
		Return(deliveredChan, &deliverErrCount)
	m.On("streamingOldmailWriteout", inDeliveredChan, oldmailPath, mock.Anything, mock.Anything).
		Return(&oldmailErrCount, nil)

	err := downloadMissingEmailsToFolder(m, maildirPath, oldmailFileName, mi)

	assert.Error(t, err)
	assert.Equal(
		t, "there were 1/1/1 errors while: retrieving/delivering/remembering mail", err.Error(),
	)
	m.AssertExpectations(t)
}

func TestDownloaderSelectFolder(t *testing.T) {
	var mbox *imap.MailboxStatus
	m := &mockClient{}
	m.On("Select", mock.Anything, mock.Anything).Return(mbox, fmt.Errorf("some error"))
	dl := &downloader{
		imapOps:    m,
		deliverOps: nil,
	}

	_, err := dl.selectFolder("some-folder")

	assert.Error(t, err)
}

func TestDownloaderGetAllMessageUUIDs(t *testing.T) {
	mbox := &imap.MailboxStatus{
		Messages: 0,
	}
	m := &mockClient{}
	m.On("Fetch", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("some error"))
	dl := &downloader{
		imapOps:    m,
		deliverOps: nil,
	}

	_, err := dl.getAllMessageUUIDs(mbox)

	assert.Error(t, err)
}

func TestDownloaderStreamingOldmailWriteout(t *testing.T) {
	dl := &downloader{}
	inChan := make(chan oldmail)
	close(inChan)
	var wg, startWg *sync.WaitGroup

	_, err := dl.streamingOldmailWriteout(inChan, "", wg, startWg)

	assert.Error(t, err)
}

func TestDownloaderStreamingRetrieval(t *testing.T) {
	m := &mockClient{}
	m.On("Fetch", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("some error"))
	dl := &downloader{
		imapOps:    m,
		deliverOps: nil,
	}
	var wg, startWg sync.WaitGroup
	interrupted := func() bool { return false }

	_, errPtr, err := dl.streamingRetrieval(nil, &wg, &startWg, interrupted)

	assert.NoError(t, err)
	wg.Wait()
	assert.Equal(t, 1, *errPtr)
}

func TestDownloaderStreamingDelivery(t *testing.T) {
	dl := &downloader{}
	inChan := make(chan emailOps)
	close(inChan)
	var wg, startWg sync.WaitGroup

	_, errPtr := dl.streamingDelivery(inChan, "", 42, &wg, &startWg)

	wg.Wait()
	assert.Equal(t, 0, *errPtr)
}
