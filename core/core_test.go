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
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockImapgrabber struct {
	mock.Mock
}

func (m *mockImapgrabber) authenticateClient(cfg IMAPConfig) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func (m *mockImapgrabber) logout(doTerminate bool) error {
	args := m.Called(doTerminate)
	return args.Error(0)
}

func (m *mockImapgrabber) getFolderList() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockImapgrabber) downloadMissingEmailsToFolder(
	maildirPath maildirPathT,
	oldmailName string,
) error {
	args := m.Called(maildirPath, oldmailName)
	return args.Error(0)
}

func setUpCoreTest(t *testing.T, m *mockImapgrabber) {
	orgNewImapgrabOps := NewImapgrabOps
	t.Cleanup(func() { NewImapgrabOps = orgNewImapgrabOps })

	NewImapgrabOps = func() ImapgrabOps {
		return m
	}
}

// Tests for Imapgrabber test the bare minimum.

func TestImapgrabberAuthenticate(t *testing.T) {
	ig, ok := NewImapgrabOps().(*Imapgrabber)
	assert.True(t, ok)
	// We cannot authenticate with empty credentials.
	err := ig.authenticateClient(IMAPConfig{})
	assert.Error(t, err)
}

func TestImapgrabberGetFolderList(t *testing.T) {
	ig, ok := NewImapgrabOps().(*Imapgrabber)
	assert.True(t, ok)

	m := setUpMockClient(t, nil, nil, nil)
	m.On("List", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("some error"))
	ig.imapOps = m

	_, err := ig.getFolderList()

	assert.Error(t, err)
}

func TestImapgrabberDownloadMissingEmails(t *testing.T) {
	ig, ok := NewImapgrabOps().(*Imapgrabber)
	assert.True(t, ok)

	m := setUpMockClient(t, nil, nil, nil)
	ig.imapOps = m

	mi := &mockInterrupter{}
	mi.On("interrupted").Return(false)
	defer mi.AssertExpectations(t)
	ig.interruptOps = mi

	err := ig.downloadMissingEmailsToFolder(maildirPathT{}, "")

	assert.Error(t, err)
}

func TestImapgrabberDowloadMissingEmailsSkipIfInterrupted(t *testing.T) {
	ig, ok := NewImapgrabOps().(*Imapgrabber)
	assert.True(t, ok)

	mi := &mockInterrupter{}
	mi.On("interrupted").Return(true)
	defer mi.AssertExpectations(t)
	ig.interruptOps = mi

	err := ig.downloadMissingEmailsToFolder(maildirPathT{}, "")

	assert.Error(t, err)
}

func TestImapgrabberLogout(t *testing.T) {
	ig, ok := NewImapgrabOps().(*Imapgrabber)
	assert.True(t, ok)

	m := setUpMockClient(t, nil, nil, nil)
	m.On("Logout").Return(fmt.Errorf("some error"))
	ig.imapOps = m

	mi := &mockInterrupter{}
	mi.On("deregister").Return()
	defer mi.AssertExpectations(t)
	ig.interruptOps = mi

	err := ig.logout(false)

	assert.Error(t, err)
}

func TestImapgrabberTerminate(t *testing.T) {
	ig, ok := NewImapgrabOps().(*Imapgrabber)
	assert.True(t, ok)

	m := setUpMockClient(t, nil, nil, nil)
	m.On("Terminate").Return(fmt.Errorf("some error"))
	ig.imapOps = m

	mi := &mockInterrupter{}
	mi.On("deregister").Return()
	defer mi.AssertExpectations(t)
	ig.interruptOps = mi

	err := ig.logout(true)

	assert.Error(t, err)
}

func TestTryConnect(t *testing.T) {
	cfg := IMAPConfig{
		Server:   "some-server",
		Port:     42,
		User:     "some user",
		Password: "this is very secret",
	}

	mock := &mockImapgrabber{}
	defer mock.AssertExpectations(t)
	mock.On("authenticateClient", cfg).Return(nil)
	mock.On("logout", false).Return(fmt.Errorf("some error"))

	setUpCoreTest(t, mock)

	err := TryConnect(cfg)

	assert.Error(t, err)
}

func TestGetAllFolders(t *testing.T) {
	cfg := IMAPConfig{
		Server:   "some-server",
		Port:     42,
		User:     "some user",
		Password: "this is very secret",
	}
	folders := []string{"f1", "f2"}

	mock := &mockImapgrabber{}
	mock.On("authenticateClient", cfg).Return(nil)
	mock.On("getFolderList").Return(folders, nil)
	mock.On("logout", false).Return(fmt.Errorf("some error"))

	setUpCoreTest(t, mock)

	actualFolders, err := GetAllFolders(cfg)

	assert.Error(t, err)
	assert.Equal(t, "some error", err.Error())
	assert.Equal(t, folders, actualFolders)
	mock.AssertExpectations(t)
}

func TestDownloadFolder(t *testing.T) {
	cfg := IMAPConfig{
		Server:   "some-server",
		Port:     42,
		User:     "some_user",
		Password: "this is very secret",
	}
	folders := []string{"f1"}
	maildir := "/some/dir"
	maildirPathF1 := maildirPathT{base: maildir, folder: "f1"}
	oldmailF1 := "oldmail-some-server-42-some_user-f1"

	mock := &mockImapgrabber{}
	mock.On("authenticateClient", cfg).Return(nil)
	mock.On("getFolderList").Return(folders, nil)
	mock.On("logout", false).Return(fmt.Errorf("some error"))
	mock.On("downloadMissingEmailsToFolder", maildirPathF1, oldmailF1).Return(nil)

	setUpCoreTest(t, mock)

	err := DownloadFolder(cfg, folders, maildir, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "some error")
	mock.AssertExpectations(t)
}

func TestDownloadFolderDownloadErr(t *testing.T) {
	cfg := IMAPConfig{
		Server:   "some-server",
		Port:     42,
		User:     "some_user",
		Password: "this is very secret",
	}
	folders := []string{"f1", "f2", "f3"}
	maildir := "/some/dir"
	maildirPathF1 := maildirPathT{base: maildir, folder: "f1"}
	maildirPathF2 := maildirPathT{base: maildir, folder: "f2"}
	oldmailF1 := "oldmail-some-server-42-some_user-f1"
	oldmailF2 := "oldmail-some-server-42-some_user-f2"

	mock := &mockImapgrabber{}
	// Only one of the logins will fail, the others will succeed. That means one goroutine will
	// download successfully while the other one will not.
	mock.On("authenticateClient", cfg).Twice().Return(nil)
	mock.On("authenticateClient", cfg).Once().Return(fmt.Errorf("some auth error"))
	mock.On("getFolderList").Return(folders, nil)
	mock.On("logout", true).Return(fmt.Errorf("some logout error"))
	mock.On("downloadMissingEmailsToFolder", maildirPathF1, oldmailF1).Return(nil)
	mock.On("downloadMissingEmailsToFolder", maildirPathF2, oldmailF2).
		Return(fmt.Errorf("some download error"))

	setUpCoreTest(t, mock)

	// We download with three goroutines. The first one succeeds apart from logout. The second one
	// fails at the download step. The third one doesn't even manage to authenticate.
	err := DownloadFolder(cfg, folders, maildir, 3)

	assert.Error(t, err)
	// We receive all errors concatenated.
	assert.Contains(t, err.Error(), "some download error")
	assert.Contains(t, err.Error(), "some logout error")
	assert.Contains(t, err.Error(), "some auth error")
	mock.AssertExpectations(t)
}

func TestDownloadFolderAuthErr(t *testing.T) {
	cfg := IMAPConfig{
		Server:   "some-server",
		Port:     42,
		User:     "some_user",
		Password: "this is very secret",
	}
	folders := []string{"f1", "f2"}
	maildir := "/some/random/dir"

	mock := &mockImapgrabber{}
	mock.On("authenticateClient", cfg).Return(fmt.Errorf("some auth error"))

	setUpCoreTest(t, mock)

	// Ensure sequential download to trigger the error reliably.
	err := DownloadFolder(cfg, folders, maildir, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "some auth error")
	mock.AssertExpectations(t)
}

func TestPartitionFolders(t *testing.T) {
	inFolders := []string{"f1", "f2", "f3", "f4"}
	threads := 3
	expectedPartitions := [][]string{{"f1", "f4"}, {"f2"}, {"f3"}}

	outPartitions := partitionFolders(inFolders, threads)

	assert.Equal(t, expectedPartitions, outPartitions)
}

func TestPartitionFoldersMoreThreadsThanFolders(t *testing.T) {
	inFolders := []string{"f1"}
	threads := 3
	expectedPartitions := [][]string{{"f1"}}

	outPartitions := partitionFolders(inFolders, threads)

	assert.Equal(t, expectedPartitions, outPartitions)
}

func TestServerMaildirInitError(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "i do not exist")
	err := ServeMaildir(IMAPConfig{}, 30123, tmp)
	assert.Error(t, err)
}

func TestServerMaildir(t *testing.T) {
	tmp := filepath.Join(t.TempDir())
	var err error

	syncChan := make(chan bool)
	go func() {
		err = ServeMaildir(IMAPConfig{}, 30123, tmp)
		syncChan <- true
	}()

	// Wait a while to ensure that the signal we send will be received by the local server and not
	// the testing process.
	time.Sleep(100 * time.Millisecond)

	signalSelf(t, os.Interrupt)
	<-syncChan

	assert.NoError(t, err)
}
