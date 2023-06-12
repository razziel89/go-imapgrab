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
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/backendutil"
	"github.com/emersion/go-imap/backend/memory"
)

type serverMailbox struct {
	maildir  maildirPathT
	messages []*serverMessage
}

// Name provids the mailboxes name.
func (mb *serverMailbox) Name() string {
	logInfo("backend mailbox name")
	return mb.maildir.folderName()
}

// Info provids some information about the maibox.
func (mb *serverMailbox) Info() (*imap.MailboxInfo, error) {
	logInfo("backend mailbox info")
	info := &imap.MailboxInfo{
		Delimiter: "/",
		Name:      mb.maildir.folderName(),
	}
	return info, nil
}

// Status provides the mailboxes status.
func (mb *serverMailbox) Status(items []imap.StatusItem) (*imap.MailboxStatus, error) {
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
	// Indicate that this is a read-only mailbox.
	status.ReadOnly = true

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
func (mb *serverMailbox) SetSubscribed(_ bool) error {
	logInfo("backend mailbox set subscribed")
	return nil
}

// Check is a no-op.
func (mb *serverMailbox) Check() error {
	logInfo("backend mailbox check")
	return nil
}

// ListMessages lists messages in a mailbox. Uids and indices are identical in this case.
func (mb *serverMailbox) ListMessages(
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
func (mb *serverMailbox) SearchMessages(_ bool, criteria *imap.SearchCriteria) ([]uint32, error) {
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
func (mb *serverMailbox) CreateMessage(_ []string, _ time.Time, _ imap.Literal) error {
	logInfo("backend create message")
	return errReadOnlyServer
}

// UpdateMessagesFlags updates message flags. We accept all flag updates unconditionally but those
// updates are not persisted. All flags are reset once the local fake IMAP server is restarted.
func (mb *serverMailbox) UpdateMessagesFlags(
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
func (mb *serverMailbox) CopyMessages(_ bool, _ *imap.SeqSet, _ string) error {
	logInfo("backend mailbox copy messages")
	return errReadOnlyServer
}

// Expunge removes messags that shall be removed.
func (mb *serverMailbox) Expunge() error {
	logInfo("backend mailbox expunge")
	return errReadOnlyServer
}

type pathAndInfo struct {
	path string
	info fs.FileInfo
}

func (mb *serverMailbox) addMessages() error {
	base := mb.maildir.folderPath()
	files := []pathAndInfo{}
	for _, dir := range []string{"new", "cur"} {
		moreFiles, err := ioutil.ReadDir(filepath.Join(base, dir))
		if err != nil {
			return err
		}
		for idx := range moreFiles {
			files = append(files, pathAndInfo{
				path: filepath.Join(base, dir, moreFiles[idx].Name()),
				info: moreFiles[idx],
			})
		}
	}
	// Sort files by modification time to get some semblance of order.
	sort.Slice(files, func(i, j int) bool {
		return files[i].info.ModTime().Before(files[j].info.ModTime())
	})

	messages := make([]*serverMessage, 0, len(files))
	for count, file := range files {
		msg := &serverMessage{
			path:   file.path,
			filled: false,
			lock:   &sync.Mutex{},
			msg: &memory.Message{
				// Identify by mod time.
				Date: file.info.ModTime(),
				Uid:  uint32(count + 1),
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
