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

// Package core provides central functionality for backing up IMAP mailboxes.
package core

import (
	"fmt"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

const (
	folderListBuffer = 10
)

// IMAPConfig is a configuration needed to access an IMAP server.
type IMAPConfig struct {
	Server   string
	Port     int
	User     string
	Password string
}

// GetAllFolders retrieves a list of all monitors in a mailbox.
func GetAllFolders(cfg IMAPConfig) (folders []string, err error) {
	if len(cfg.Password) == 0 {
		logError("empty password detected")
		err = fmt.Errorf("password not set")
		return
	}

	logInfo(fmt.Sprintf("connecting to server %s", cfg.Server))
	serverWithPort := fmt.Sprintf("%s:%d", cfg.Server, cfg.Port)
	var imapClient *client.Client
	if imapClient, err = client.DialTLS(serverWithPort, nil); err != nil {
		logError("cannot connect")
		return
	}
	logInfo("connected")

	logInfo(fmt.Sprintf("logging in as %s with provided password", cfg.User))
	if err = imapClient.Login(cfg.User, cfg.Password); err != nil {
		logError("cannot log in")
		return
	}
	logInfo("logged in")

	// Make sure to log out in the end if we logged in successfully.
	defer func() {
		// Don't overwrite the error if it has already been set.
		if logoutErr := imapClient.Logout(); logoutErr != nil && err == nil {
			err = logoutErr
		}
	}()

	logInfo("retrieving folders")
	mailboxes := make(chan *imap.MailboxInfo, folderListBuffer)
	go func() {
		err = imapClient.List("", "*", mailboxes)
	}()
	for m := range mailboxes {
		folders = append(folders, m.Name)
	}
	logInfo("retrieved folders")
	return
}
