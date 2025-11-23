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
	"errors"
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
)

var errReadOnlyServer = errors.New("cannot execute action, this is a read-only IMAP server")

// serverBackend is the v2 IMAP server backend
type serverBackend struct {
	path     string
	username string
	password string
	user     *serverUser
}

// NewSession creates a new IMAP session for v2 server
func (b *serverBackend) NewSession(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
	sess := &serverSession{
		backend: b,
	}
	return sess, &imapserver.GreetingData{}, nil
}

// serverSession implements imapserver.Session for v2
type serverSession struct {
	backend  *serverBackend
	user     *serverUser
	mailbox  *serverMailbox
}

func (sess *serverSession) Close() error {
	return nil
}

func (sess *serverSession) Login(username, password string) error {
	logInfo(fmt.Sprintf("attempting to log in as %s", username))
	if username != sess.backend.username {
		logInfo(fmt.Sprintf("login as %s failed, bad user", username))
		return imapserver.ErrAuthFailed
	}
	if password != sess.backend.password {
		logInfo(fmt.Sprintf("login as %s failed, bad password", username))
		return imapserver.ErrAuthFailed
	}
	logInfo(fmt.Sprintf("login as %s succeeded", username))
	sess.user = sess.backend.user
	return nil
}

// Authenticated state methods
func (sess *serverSession) Select(mailbox string, options *imap.SelectOptions) (*imap.SelectData, error) {
	logInfo(fmt.Sprintf("backend select mailbox %s", mailbox))
	// Find the mailbox
	for _, mbox := range sess.user.mailboxes {
		if mbox.maildir.folderName() == mailbox {
			sess.mailbox = mbox
			return &imap.SelectData{
				Flags:       []imap.Flag{imap.FlagSeen},
				NumMessages: uint32(len(mbox.messages)),
				UIDValidity: 1,
				UIDNext:     imap.UID(len(mbox.messages) + 1),
			}, nil
		}
	}
	return nil, fmt.Errorf("mailbox not found")
}

func (sess *serverSession) Create(_ string, _ *imap.CreateOptions) error {
	return errReadOnlyServer
}

func (sess *serverSession) Delete(_ string) error {
	return errReadOnlyServer
}

func (sess *serverSession) Rename(_, _ string, _ *imap.RenameOptions) error {
	return errReadOnlyServer
}

func (sess *serverSession) Subscribe(_ string) error {
	return nil // no-op for read-only server
}

func (sess *serverSession) Unsubscribe(_ string) error {
	return nil // no-op for read-only server
}

func (sess *serverSession) List(w *imapserver.ListWriter, ref string, patterns []string, options *imap.ListOptions) error {
	logInfo("backend list mailboxes")
	for _, mbox := range sess.user.mailboxes {
		data := &imap.ListData{
			Mailbox: mbox.maildir.folderName(),
		}
		if err := w.WriteList(data); err != nil {
			return err
		}
	}
	return nil
}

func (sess *serverSession) Status(mailbox string, options *imap.StatusOptions) (*imap.StatusData, error) {
	logInfo(fmt.Sprintf("backend status mailbox %s", mailbox))
	for _, mbox := range sess.user.mailboxes {
		if mbox.maildir.folderName() == mailbox {
			data := &imap.StatusData{
				Mailbox:     mailbox,
				UIDNext:     imap.UID(len(mbox.messages) + 1),
				UIDValidity: 1,
			}
			numMessages := uint32(len(mbox.messages))
			data.NumMessages = &numMessages
			return data, nil
		}
	}
	return nil, fmt.Errorf("mailbox not found")
}

func (sess *serverSession) Append(_ string, _ imap.LiteralReader, _ *imap.AppendOptions) (*imap.AppendData, error) {
	return nil, errReadOnlyServer
}

func (sess *serverSession) Poll(_ *imapserver.UpdateWriter, _ bool) error {
	return nil // no-op for read-only server
}

func (sess *serverSession) Idle(_ *imapserver.UpdateWriter, _ <-chan struct{}) error {
	return nil // no-op for read-only server
}

// Selected state methods
func (sess *serverSession) Unselect() error {
	sess.mailbox = nil
	return nil
}

func (sess *serverSession) Expunge(_ *imapserver.ExpungeWriter, _ *imap.UIDSet) error {
	return errReadOnlyServer
}

func (sess *serverSession) Search(_ imapserver.NumKind, _ *imap.SearchCriteria, _ *imap.SearchOptions) (*imap.SearchData, error) {
	// Minimal search implementation - return empty results
	return &imap.SearchData{}, nil
}

func (sess *serverSession) Fetch(w *imapserver.FetchWriter, numSet imap.NumSet, options *imap.FetchOptions) error {
	if sess.mailbox == nil {
		return fmt.Errorf("no mailbox selected")
	}
	
	logInfo("backend fetch messages")
	// For now, return a minimal implementation that doesn't actually fetch messages
	// A full implementation would iterate through messages and write fetch data
	return nil
}

func (sess *serverSession) Store(_ *imapserver.FetchWriter, _ imap.NumSet, _ *imap.StoreFlags, _ *imap.StoreOptions) error {
	return errReadOnlyServer
}

func (sess *serverSession) Copy(_ imap.NumSet, _ string) (*imap.CopyData, error) {
	return nil, errReadOnlyServer
}

func (b *serverBackend) addUser() error {
	user := &serverUser{name: b.username}
	b.user = user
	err := user.addMailboxes(b.path)
	return err
}

func newBackend(path, username, password string) (*serverBackend, error) {
	bcknd := &serverBackend{
		path:     path,
		username: username,
		password: password,
	}
	err := bcknd.addUser()
	return bcknd, err
}
