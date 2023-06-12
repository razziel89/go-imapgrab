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

	"github.com/emersion/go-imap/backend"
)

type serverUser struct {
	name      string
	mailboxes []*serverMailbox
}

// Username provides the user's name.
func (u *serverUser) Username() string {
	logInfo("backend username")
	return u.name
}

// ListMailboxes lists a mailbox.
func (u *serverUser) ListMailboxes(_ bool) ([]backend.Mailbox, error) {
	logInfo("backend list mailboxes")
	boxes := []backend.Mailbox{}
	for idx := range u.mailboxes {
		boxes = append(boxes, u.mailboxes[idx])
	}
	logInfo(fmt.Sprintf("listed %d mailboxes", len(boxes)))
	return boxes, nil
}

// GetMailbox retrieves a mailbox.
func (u *serverUser) GetMailbox(name string) (backend.Mailbox, error) {
	logInfo(fmt.Sprintf("backend get mailbox %s", name))
	for _, mailbox := range u.mailboxes {
		if mailbox.maildir.folderName() == name {
			return mailbox, nil
		}
	}
	return nil, fmt.Errorf("unknown mailbox %s", name)
}

// CreateMailbox creates a mailbox.
func (u *serverUser) CreateMailbox(_ string) error {
	logInfo("backend create mailbox")
	return errReadOnlyServer
}

// DeleteMailbox deletes a mailbox.
func (u *serverUser) DeleteMailbox(_ string) error {
	logInfo("backend delete mailbox")
	return errReadOnlyServer
}

// RenameMailbox renames a mailbox.
func (u *serverUser) RenameMailbox(_, _ string) error {
	logInfo("backend rename mailbox")
	return errReadOnlyServer
}

// Logout logs out the user. This is a no-op.
func (u *serverUser) Logout() error {
	logInfo("backend logout")
	return nil
}

func (u *serverUser) addMailboxes(path string) error {
	dirs, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	boxes := []*serverMailbox{}
	for _, dir := range dirs {
		maildirPath := maildirPathT{base: path, folder: dir.Name()}
		if dir.IsDir() && isMaildir(maildirPath.folderPath()) {
			box := &serverMailbox{maildir: maildirPath}
			boxes = append(boxes, box)
		}
	}

	u.mailboxes = boxes

	errs := threadSafeErrors{}
	for _, box := range boxes {
		err := box.addMessages()
		errs.add(err)
	}
	logInfo(fmt.Sprintf("readin in %d mailboxes", len(boxes)))

	return errs.err()
}
