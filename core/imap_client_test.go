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
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockClient struct {
	mock.Mock

	mailboxes []*imap.MailboxInfo
	messages  []*imap.Message
}

func (mc *mockClient) Login(username string, password string) error {
	args := mc.Called(username, password)
	return args.Error(0)
}

func (mc *mockClient) List(ref string, name string, ch chan *imap.MailboxInfo) error {
	defer close(ch)
	args := mc.Called(ref, name, ch)
	for _, box := range mc.mailboxes {
		ch <- box
	}
	return args.Error(0)
}

func (mc *mockClient) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	args := mc.Called(name, readOnly)
	return args.Get(0).(*imap.MailboxStatus), args.Error(1)
}

func (mc *mockClient) Fetch(
	seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message,
) error {
	defer close(ch)
	args := mc.Called(seqset, items, ch)
	for _, msg := range mc.messages {
		ch <- msg
	}
	return args.Error(0)
}

func (mc *mockClient) Logout() error {
	args := mc.Called()
	return args.Error(0)
}

func (mc *mockClient) Terminate() error {
	args := mc.Called()
	return args.Error(0)
}

func setUpMockClient(
	t *testing.T, boxes []*imap.MailboxInfo, messages []*imap.Message, err error,
) *mockClient {
	mock := &mockClient{
		mailboxes: boxes,
		messages:  messages,
	}
	orgClientGetter := newImapClient
	newImapClient = func(addr string) (imapOps, error) {
		return mock, err
	}
	t.Cleanup(func() { newImapClient = orgClientGetter })
	t.Cleanup(func() { mock.AssertExpectations(t) })
	return mock
}

func TestAuthFailure(t *testing.T) {
	_, err := newImapClient("")
	assert.Error(t, err)
}

func TestAuthenticateClientSuccess(t *testing.T) {
	mock := setUpMockClient(t, nil, nil, nil)
	mock.On("Login", "someone", "some password").Return(nil)

	config := IMAPConfig{
		User:     "someone",
		Password: "some password",
	}

	client, err := authenticateClient(config)

	assert.NoError(t, err)
	assert.Equal(t, client, mock)
}

func TestAuthenticateClientNoPassword(t *testing.T) {
	_ = setUpMockClient(t, nil, nil, nil)
	config := IMAPConfig{}

	_, err := authenticateClient(config)

	assert.Error(t, err)
}

func TestAuthenticateClientCannotConnect(t *testing.T) {
	loginErr := fmt.Errorf("cannot log in")
	_ = setUpMockClient(t, nil, nil, loginErr)

	config := IMAPConfig{
		User:     "someone",
		Password: "some password",
	}

	_, err := authenticateClient(config)

	assert.Error(t, err)
	assert.Equal(t, loginErr, err)
}

func TestAuthenticateClientWrongCredentials(t *testing.T) {
	loginErr := fmt.Errorf("wrong credentials")
	mock := setUpMockClient(t, nil, nil, nil)
	mock.On("Login", "someone", "wrong password").Return(loginErr)

	config := IMAPConfig{
		User:     "someone",
		Password: "wrong password",
	}

	_, err := authenticateClient(config)

	assert.Error(t, err)
}

func TestGetFolderListSuccess(t *testing.T) {
	boxes := []*imap.MailboxInfo{
		{Name: "b1"},
		{Name: "b2"},
		{Name: "b3"},
	}
	m := setUpMockClient(t, boxes, nil, nil)
	m.On("List", "", "*", mock.Anything).Return(nil)

	list, err := getFolderList(m)

	assert.NoError(t, err)
	assert.Equal(t, []string{"b1", "b2", "b3"}, list)
}

func TestGetFolderListError(t *testing.T) {
	listErr := fmt.Errorf("list error")
	boxes := []*imap.MailboxInfo{
		{Name: "b1"},
	}
	m := setUpMockClient(t, boxes, nil, nil)
	m.On("List", "", "*", mock.Anything).Return(listErr)

	_, err := getFolderList(m)

	assert.Error(t, err)
	assert.Equal(t, listErr, err)
}

func TestSelectFolderSuccess(t *testing.T) {
	expectedStatus := &imap.MailboxStatus{Messages: 42}
	m := setUpMockClient(t, nil, nil, nil)
	m.On("Select", "some folder", true).Return(expectedStatus, nil)

	status, err := selectFolder(m, "some folder")

	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, status)
}

func TestStreamingRetrievalSuccess(t *testing.T) {
	status := &imap.MailboxStatus{
		Messages:    16,
		UidValidity: 42,
	}
	ranges := []rangeT{
		{start: 10, end: 11},
		{start: 12, end: 13},
		{start: 16, end: 17},
	}
	messages := []*imap.Message{
		{Uid: 10},
		{Uid: 12},
		{Uid: 16},
	}

	expectedSeqSet := &imap.SeqSet{}
	expectedSeqSet.AddNum(10, 12, 16)
	expectedFetchRequest := []imap.FetchItem{
		imap.FetchUid, imap.FetchInternalDate, imap.FetchRFC822,
	}

	m := setUpMockClient(t, nil, messages, nil)
	m.On("Fetch", expectedSeqSet, expectedFetchRequest, mock.Anything).Return(
		fmt.Errorf("retrieval error"),
	)

	var wg, stwg sync.WaitGroup
	stwg.Add(1)
	interrupted := func() bool { return false }

	emailChan, errPtr, err := streamingRetrieval(status, m, ranges, &wg, &stwg, interrupted)

	assert.NoError(t, err)
	assert.Zero(t, *errPtr)

	// Wait a while and check that nothing has happened yet.
	select {
	case <-emailChan:
		// Fail if something has happened yet.
		t.Fail()
	case <-time.After(time.Millisecond * 100): // nolint: gomnd
		// Continue if nothing has happened yet.
	}

	// Actually trigger operations and read from output channel.
	stwg.Done()
	emails := []*imap.Message{}
	for em := range emailChan {
		// Convert type back for easier comparison.
		msg := em.(*imap.Message)
		emails = append(emails, msg)
	}
	wg.Wait()

	assert.Equal(t, 1, *errPtr)
	assert.Equal(t, messages, emails)
}

func TestStreamingRetrievalError(t *testing.T) {
	status := &imap.MailboxStatus{}
	m := setUpMockClient(t, nil, nil, nil)

	// These ranges trigger an initial error.
	ranges := []rangeT{
		{start: 20, end: 10},
	}

	var wg, stwg sync.WaitGroup
	stwg.Add(1)
	interrupted := func() bool { return false }

	_, _, err := streamingRetrieval(status, m, ranges, &wg, &stwg, interrupted)

	assert.Error(t, err)
}

func TestStreamingRetrievalInterrupt(t *testing.T) {
	status := &imap.MailboxStatus{
		Messages:    16,
		UidValidity: 42,
	}
	ranges := []rangeT{{start: 10, end: 11}}
	messages := []*imap.Message{}

	m := &mockClient{messages: messages}
	m.On("Fetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// This code was taken from setUpMockClient because we do not want to assert expectations here.
	orgClientGetter := newImapClient
	newImapClient = func(addr string) (imapOps, error) {
		return m, nil
	}
	t.Cleanup(func() { newImapClient = orgClientGetter })

	var wg, stwg sync.WaitGroup
	stwg.Add(1)

	// Return true, which simulates the receipt of a signal for interruption, which will trigger the
	// interrupt case. Interrupts are handled preferentially compared to message conversion.
	interrupted := func() bool { return true }

	_, errPtr, err := streamingRetrieval(status, m, ranges, &wg, &stwg, interrupted)

	assert.NoError(t, err)

	// We will not have to call stwg.Done() because the interrupt handler goroutine will
	// automatically call wg.Done() once it has handled the interrupt.
	wg.Wait()
	assert.Equal(t, 1, *errPtr)

	// We call stwg.Done() here to give all goroutines the chance to finish execution.
	stwg.Done()
}

func TestUIDToStrng(t *testing.T) {
	u := uid{Mbox: 42, Message: 10}
	str := "42/10"

	assert.Equal(t, str, fmt.Sprint(u))
}

func TestGetAllMessageUUIDsSuccess(t *testing.T) {
	status := &imap.MailboxStatus{
		Messages:    3,
		UidValidity: 42,
	}
	messages := []*imap.Message{
		{Uid: 10},
		// There are no guarantees the server does not return nil. Thus, we make sure to ignore such
		// values.
		nil,
		{Uid: 12},
		nil,
		{Uid: 16},
	}

	expectedSeqSet := &imap.SeqSet{}
	expectedSeqSet.AddRange(1, 3)
	expectedUUIDs := []uid{
		{Mbox: 42, Message: 10},
		{Mbox: 42, Message: 12},
		{Mbox: 42, Message: 16},
	}
	expectedFetchRequest := []imap.FetchItem{imap.FetchUid, imap.FetchInternalDate}

	m := setUpMockClient(t, nil, messages, nil)
	m.On("Fetch", expectedSeqSet, expectedFetchRequest, mock.Anything).Return(nil)

	uids, err := getAllMessageUUIDs(status, m)

	assert.NoError(t, err)
	assert.Equal(t, expectedUUIDs, uids)
}
