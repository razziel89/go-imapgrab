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
	"os"
	"path/filepath"
)

// IMAPConfig is a configuration needed to access an IMAP server.
type IMAPConfig struct {
	Server   string
	Port     int
	User     string
	Password string
}

// GetAllFolders retrieves a list of all folders in a mailbox.
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

// DownloadFolder downloads all not yet downloaded email from a folder in a mailbox to a maildir.
// The oldmail file in the parent directory of the maildir is used to determine which emails have
// already been downloaded. According to the [maildir specs](https://cr.yp.to/proto/maildir.html),
// the email is first downloaded into the `tmp` sub-directory and then moved atomically to the `new`
// sub-directory.
///nolint:funlen
func DownloadFolder(cfg IMAPConfig, folder, maildirPath string) error {
	// Retrieve information about emails that have already been downloaded.
	oldmails, oldmailPath, err := initExistingMaildir(cfg, maildirPath)
	if err != nil {
		return err
	}

	// Authenticate against the remote server.
	imapClient, err := authenticateClient(cfg)
	if err != nil {
		return err
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
		return err
	}
	uidvalidity := int(mbox.UidValidity)

	// Retrieve information about which emails are present on the remote system and check which ones
	// are missing when comparing against those on disk.
	uids, err := getAllMessageUUIDs(mbox, imapClient)
	if err != nil {
		return err
	}
	logInfo(fmt.Sprintf("received information for %d emails", len(uids)))
	fmt.Println(uids)
	missingIDRanges, err := determineMissingIDs(oldmails, uids)
	if err != nil {
		return err
	}
	logInfo(fmt.Sprintf("will download %d new emails", len(missingIDRanges)))
	fmt.Println(missingIDRanges)

	// Download missing emails and store them on disk.
	for _, missingRange := range missingIDRanges {
		msgs, err := getMessageRange(mbox, imapClient, missingRange)
		if err != nil {
			return err
		}
		// Deliver each email to the `tmp` directory and move them to the `new` directory.
		for _, msg := range msgs {
			text, oldmail, err := rfc822FromEmail(msg, uidvalidity)
			if err != nil {
				return err
			}
			logInfo(fmt.Sprintf("downloaded email %s", oldmail))
			fileName, err := newUniqueName()
			if err != nil {
				return err
			}
			tmpPath := filepath.Join(maildirPath, tmpMaildir, fileName)
			newPath := filepath.Join(maildirPath, newMaildir, fileName)
			if isFile(tmpPath) {
				return fmt.Errorf("unique file name '%s' is not unique", tmpPath)
			}
			logInfo(fmt.Sprintf("writing new email to file %s", tmpPath))
			err = os.WriteFile(tmpPath, []byte(text), 0644) //nolint:gosec,gomnd
			if err != nil {
				return err
			}
			oldmails = append(oldmails, oldmail)
			logInfo(fmt.Sprintf("moving email to permanent storage location %s", newPath))
			if isFile(newPath) {
				return fmt.Errorf("permanent storate '%s' already exists", newPath)
			}
			err = os.Rename(tmpPath, newPath)
			if err != nil {
				return err
			}
		}
	}

	// Write out information about newly retrieved emails.
	logInfo(fmt.Sprintf("writing oldmail file %s", oldmailPath))
	if err := writeOldmail(oldmails, oldmailPath); err != nil {
		return err
	}
	logInfo("wrote new oldmail file")

	return nil
}
