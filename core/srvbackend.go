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

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
)

const readOnlyServerErr = "cannot execute action, this is a read-only IMAP server"

type igrabBackend struct {
	path     string
	username string
	password string
	user     *igrabUser
}

func (b *igrabBackend) Login(_ *imap.ConnInfo, username, password string) (backend.User, error) {
	logInfo(fmt.Sprintf("attempting to log in as %s", username))
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
