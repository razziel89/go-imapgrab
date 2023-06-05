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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/backend/backendutil"
	"github.com/emersion/go-imap/backend/memory"
)

const readOnlyServerErr = "cannot execute action, this is a read-only IMAP server"

type igrabBackend struct {
	path     string
	username string
	password string
	user     *igrabUser
}

func (b *igrabBackend) Login(_ *imap.ConnInfo, username, password string) (backend.User, error) {
	logInfo(fmt.Sprintf("attempting to log in as %s with password %s", username, password))
	if username != b.username {
		logInfo(fmt.Sprintf("login as %s failed, bad user", username))
		return nil, fmt.Errorf("bad username or password")
	}
	if password != b.password {
		logInfo(fmt.Sprintf("login as %s failed, bad password", username))
		return nil, fmt.Errorf("bad username or password")
	}
	logInfo(fmt.Sprintf("login as %s succeeded", username))
	return b.user, nil
}

type igrabUser struct {
	name      string
	mailboxes []*igrabMailbox
}

// Username provides the user's name.
func (u *igrabUser) Username() string {
	logInfo("backend username")
	return u.name
}

// ListMailboxes lists a mailbox.
func (u *igrabUser) ListMailboxes(_ bool) ([]backend.Mailbox, error) {
	logInfo("backend list mailboxes")
	boxes := []backend.Mailbox{}
	for _, mailbox := range u.mailboxes {
		mailbox := mailbox
		boxes = append(boxes, mailbox)
	}
	logInfo(fmt.Sprintf("listed %d mailboxes", len(boxes)))
	return boxes, nil
}

// GetMailbox retrieves a mailbox.
func (u *igrabUser) GetMailbox(name string) (backend.Mailbox, error) {
	logInfo(fmt.Sprintf("backend get mailbox %s", name))
	for _, mailbox := range u.mailboxes {
		if mailbox.maildir.folderName() == name {
			return mailbox, nil
		}
	}
	return nil, fmt.Errorf("unknown mailbox %s", name)
}

// CreateMailbox creates a mailbox.
func (u *igrabUser) CreateMailbox(_ string) error {
	logInfo("backend create mailbox")
	return nil
}

// DeleteMailbox deletes a mailbox.
func (u *igrabUser) DeleteMailbox(_ string) error {
	logInfo("backend delete mailbox")
	return fmt.Errorf(readOnlyServerErr)
}

// RenameMailbox renames a mailbox.
func (u *igrabUser) RenameMailbox(_, _ string) error {
	logInfo("backend rename mailbox")
	return fmt.Errorf(readOnlyServerErr)
}

// Logout logs out the user. This is a no-op.
func (u *igrabUser) Logout() error {
	logInfo("backend logout")
	return nil
}

type igrabMailbox struct {
	maildir  maildirPathT
	messages []*igrabMessage
}

// Name provids the mailboxes name.
func (mb *igrabMailbox) Name() string {
	logInfo("backend mailbox name")
	return mb.maildir.folderName()
}

// Info provids some information about the maibox.
func (mb *igrabMailbox) Info() (*imap.MailboxInfo, error) {
	logInfo("backend mailbox info")
	info := &imap.MailboxInfo{
		Delimiter: "/",
		Name:      mb.maildir.folderName(),
	}
	return info, nil
}

// Status provides the mailboxes status.
func (mb *igrabMailbox) Status(items []imap.StatusItem) (*imap.MailboxStatus, error) {
	// Determine mailbox flags first, which basically means all flags that at least one email has
	// set.
	allFlags := make(map[string]bool)
	for _, msg := range mb.messages {
		for _, flag := range msg.msg.Flags {
			allFlags[flag] = true
		}
	}
	flags := make([]string, 0, len(allFlags))
	for flag := range allFlags {
		flags = append(flags, flag)
	}

	status := imap.NewMailboxStatus(mb.maildir.folderName(), items)
	status.Flags = flags
	status.PermanentFlags = []string{"\\*"}
	// State that all messages have already been seen.
	status.UnseenSeqNum = 0

	// Determine the next unseen UID. We just assume that a message's index is identical to its UID.

	for _, name := range items {
		switch name {
		case imap.StatusMessages:
			status.Messages = uint32(len(mb.messages))
		case imap.StatusUidNext:
			status.UidNext = uint32(len(mb.messages)) + 1
		case imap.StatusUidValidity:
			status.UidValidity = 1
		case imap.StatusRecent:
			status.Recent = 0
		case imap.StatusUnseen:
			status.Unseen = 0
		case imap.StatusAppendLimit:
			status.AppendLimit = 0
		}
	}

	return status, nil
}

// SetSubscribed marks a mailbox as subscribed. We ignore that and always return all mailboxes.
func (mb *igrabMailbox) SetSubscribed(_ bool) error {
	logInfo("backend mailbox set subscribed")
	return nil
}

// Check is a no-op.
func (mb *igrabMailbox) Check() error {
	logInfo("backend mailbox check")
	return nil
}

// ListMessages lists messages in a mailbox. Uids and indices are identical in this case.
func (mb *igrabMailbox) ListMessages(
	_ bool, seqset *imap.SeqSet, items []imap.FetchItem, msgChan chan<- *imap.Message,
) error {
	logInfo("backend mailbox list messages")
	defer close(msgChan)
	for count, msg := range mb.messages {
		uidAndIdx := uint32(count + 1)
		if seqset.Contains(uidAndIdx) {
			fetched, err := msg.Fetch(uidAndIdx, items)
			if err == nil {
				msgChan <- fetched
			} else {
				logError(fmt.Sprintf("cannot fetch message: %s", err.Error()))
			}
		}
	}
	return nil
}

// SearchMessages searches for a message.
func (mb *igrabMailbox) SearchMessages(_ bool, criteria *imap.SearchCriteria) ([]uint32, error) {
	logInfo("backend mailbox search messages")
	var foundIDs []uint32
	for count, msg := range mb.messages {
		uidAndIdx := uint32(count + 1)
		ok, err := msg.Match(uidAndIdx, criteria)
		if err == nil {
			if ok {
				foundIDs = append(foundIDs, uidAndIdx)
			}
		} else {
			logError(fmt.Sprintf("cannot match message: %s", err.Error()))
		}
	}
	return foundIDs, nil
}

// CreateMessage creates a new message.
func (mb *igrabMailbox) CreateMessage(_ []string, _ time.Time, _ imap.Literal) error {
	logInfo("backend create message")
	return fmt.Errorf(readOnlyServerErr)
}

// UpdateMessagesFlags updats message flags.
func (mb *igrabMailbox) UpdateMessagesFlags(
	_ bool, seqset *imap.SeqSet, operation imap.FlagsOp, flags []string,
) error {
	for count, msg := range mb.messages {
		uidAndIdx := uint32(count + 1)
		if seqset.Contains(uidAndIdx) {
			msg.msg.Flags = backendutil.UpdateFlags(msg.msg.Flags, operation, flags)
		}
	}
	return nil
}

// CopyMessages copies a message from one mailbox to another one.
func (mb *igrabMailbox) CopyMessages(_ bool, _ *imap.SeqSet, _ string) error {
	logInfo("backend mailbox copy messages")
	return fmt.Errorf(readOnlyServerErr)
}

// Expunge removes messags that shall be removed.
func (mb *igrabMailbox) Expunge() error {
	logInfo("backend mailbox expunge")
	return fmt.Errorf(readOnlyServerErr)
}

type pathAndDate struct {
	path string
	date time.Time
}

func (mb *igrabMailbox) readMessages() error {
	base := mb.maildir.folderPath()
	newFiles, newErr := ioutil.ReadDir(filepath.Join(base, "new"))
	curFiles, curErr := ioutil.ReadDir(filepath.Join(base, "cur"))
	if newErr != nil || curErr != nil {
		return errors.Join(newErr, curErr)
	}
	// Sort files by modification time to get some semblance of order.
	sort.Slice(newFiles, func(i, j int) bool {
		return newFiles[i].ModTime().Before(newFiles[j].ModTime())
	})
	sort.Slice(curFiles, func(i, j int) bool {
		return curFiles[i].ModTime().Before(curFiles[j].ModTime())
	})

	files := make([]pathAndDate, 0, len(newFiles)+len(curFiles))
	for _, file := range curFiles {
		files = append(files, pathAndDate{
			path: filepath.Join(base, "cur", file.Name()),
			date: file.ModTime(),
		})
	}
	for _, file := range newFiles {
		files = append(files, pathAndDate{
			path: filepath.Join(base, "new", file.Name()),
			date: file.ModTime(),
		})
	}

	messages := make([]*igrabMessage, 0, len(files))
	for count, file := range files {
		msg := &igrabMessage{
			path:   file.path,
			filled: false,
			lock:   &sync.Mutex{},
			msg: &memory.Message{
				Date: file.date,
				// Identify by mod time.
				Uid: uint32(count + 1),
				// Assume all have been seen already.
				Flags: []string{"\\Seen"},
				// Size and Body will be filled in later and only on demand.
			},
		}
		messages = append(messages, msg)
	}
	logInfo(fmt.Sprintf("read %d messags for mailbox %s", len(messages), mb.maildir.folderName()))
	mb.messages = messages
	return nil
}

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

func newBackend(path, username, password string) (backend.Backend, error) {
	dirs, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	mailboxes := []*igrabMailbox{}
	for _, dir := range dirs {
		maildirPath := maildirPathT{base: path, folder: dir.Name()}
		if dir.IsDir() && isMaildir(maildirPath.folderPath()) {
			mailboxes = append(mailboxes, &igrabMailbox{maildir: maildirPath})
		}
	}

	logInfo(fmt.Sprintf("readin in %d mailboxes", len(mailboxes)))

	for _, box := range mailboxes {
		err := box.readMessages()
		if err != nil {
			return nil, err
		}
	}

	return &igrabBackend{
		path:     path,
		username: username,
		password: password,
		user: &igrabUser{
			name:      username,
			mailboxes: mailboxes,
		},
	}, nil
}
