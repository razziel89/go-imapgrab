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
	"testing"

	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockClient struct {
	mock.Mock

	mailboxes []*imap.MailboxInfo
}

func (mc *mockClient) Login(username string, password string) error {
	args := mc.Called(username, password)
	return args.Error(0)
}

func (mc *mockClient) List(ref string, name string, ch chan *imap.MailboxInfo) error {
	args := mc.Called(ref, name, ch)
	for _, box := range mc.mailboxes {
		ch <- box
	}
	close(ch)
	return args.Error(0)
}

func (mc *mockClient) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	args := mc.Called(name, readOnly)
	return args.Get(0).(*imap.MailboxStatus), args.Error(1)
}

func (mc *mockClient) Fetch(
	seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message,
) error {
	args := mc.Called(seqset, items, ch)
	return args.Error(0)
}

func (mc *mockClient) Logout() error {
	panic("not implemented") // TODO: Implement
}

func setUpMockClient(t *testing.T, boxes []*imap.MailboxInfo, err error) *mockClient {
	mock := &mockClient{
		mailboxes: boxes,
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
	mock := setUpMockClient(t, nil, nil)
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
	_ = setUpMockClient(t, nil, nil)
	config := IMAPConfig{}

	_, err := authenticateClient(config)

	assert.Error(t, err)
}

func TestAuthenticateClientCannotConnect(t *testing.T) {
	loginErr := fmt.Errorf("cannot log in")
	_ = setUpMockClient(t, nil, loginErr)

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
	mock := setUpMockClient(t, nil, nil)
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
		&imap.MailboxInfo{Name: "b1"},
		&imap.MailboxInfo{Name: "b2"},
		&imap.MailboxInfo{Name: "b3"},
	}
	m := setUpMockClient(t, boxes, nil)
	m.On("List", "", "*", mock.Anything).Return(nil)

	list, err := getFolderList(m)

	assert.NoError(t, err)
	assert.Equal(t, []string{"b1", "b2", "b3"}, list)
}

func TestGetFolderListError(t *testing.T) {
	listErr := fmt.Errorf("list error")
	boxes := []*imap.MailboxInfo{
		&imap.MailboxInfo{Name: "b1"},
	}
	m := setUpMockClient(t, boxes, nil)
	m.On("List", "", "*", mock.Anything).Return(listErr)

	_, err := getFolderList(m)

	assert.Error(t, err)
	assert.Equal(t, listErr, err)
}

func TestSelectFolderSuccess(t *testing.T) {
	expectedStatus := &imap.MailboxStatus{Messages: 42}
	m := setUpMockClient(t, nil, nil)
	m.On("Select", "some folder", true).Return(expectedStatus, nil)

	status, err := selectFolder(m, "some folder")

	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, status)
}
