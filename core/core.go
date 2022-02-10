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
	"time"
)

const (
	rfc822ExpectedNumFields = 6
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
	imapClient, err := authenticateClient(cfg)
	if err != nil {
		return
	}
	// Make sure to log out in the end if we logged in successfully.
	defer func() {
		// Don't overwrite the error if it has already been set.
		if logoutErr := imapClient.Logout(); logoutErr != nil && err == nil {
			err = logoutErr
		}
	}()

	return getFolderList(imapClient)
}

// PrintEmail reads a single email with index `idx` (1 is most recent) from a single folder `folder`
// and returns its content. This functionality will likely be removed later but it is useful for
// development.
func PrintEmail(cfg IMAPConfig, folder string, index int) (content string, err error) {
	imapClient, err := authenticateClient(cfg)
	if err != nil {
		return
	}
	// Make sure to log out in the end if we logged in successfully.
	defer func() {
		// Don't overwrite the error if it has already been set.
		if logoutErr := imapClient.Logout(); logoutErr != nil && err == nil {
			err = logoutErr
		}
	}()

	mbox, err := selectFolder(imapClient, folder)
	if err != nil {
		return
	}
	msg, err := getNthMessage(mbox, imapClient, index)
	if err != nil {
		return
	}

	fields := msg.Format()
	if len(fields) != rfc822ExpectedNumFields {
		return "", fmt.Errorf("cannot extract required RFC822 fields from email")
	}

	email := Email{}
	for _, field := range fields {
		if err := email.set(field); err != nil {
			return "", fmt.Errorf("cannot extract email data: %s", err.Error())
		}
	}
	if !email.validate() {
		return "", fmt.Errorf("cannot extract full email from reply")
	}

	return fmt.Sprint(email), nil
}

// GetAllUIDsAndTimestampts obtains all UIDs of all emails in a mailbox and their timestamps. UIDs
// are not checked for uniqueness. The time at any one index corresponds to the UID at the same
// index. This functionality will likely be removed later but it is useful for development.
func GetAllUIDsAndTimestampts(
	cfg IMAPConfig, folder string,
) (uids []int, times []time.Time, err error) {
	imapClient, err := authenticateClient(cfg)
	if err != nil {
		return
	}
	// Make sure to log out in the end if we logged in successfully.
	defer func() {
		// Don't overwrite the error if it has already been set.
		if logoutErr := imapClient.Logout(); logoutErr != nil && err == nil {
			err = logoutErr
		}
	}()

	mbox, err := selectFolder(imapClient, folder)
	if err != nil {
		return
	}
	return getAllMessageUUIDsAndTimestamps(mbox, imapClient)
}
