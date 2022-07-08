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
)

// IMAPConfig is a configuration needed to access an IMAP server.
type IMAPConfig struct {
	Server   string
	Port     int
	User     string
	Password string
}

// ImapgrabOps provides functionality for interacting with the basic imapgrab functionality such as
// listing contents of mailboxes or downloading emails.
type ImapgrabOps interface {
	// authenticateClient is used to authenticate against a remote server
	authenticateClient(IMAPConfig) error
	// logout is used to log out from an authenticated session
	logout(bool) error
	// getFolderList provides all folders in the configured mailbox
	getFolderList() ([]string, error)
	// downloadMissingEmailsToFolder downloads all emails to a local path that are present remotely
	// but missing locally
	downloadMissingEmailsToFolder(maildirPathT, string) error
}

// Imapgrabber is the defailt implementation of ImapgrabOps.
type Imapgrabber struct {
	downloadOps  downloadOps
	imapOps      imapOps
	interruptOps interruptOps
}

// authenticateClient is used to authenticate against a remote server
func (ig *Imapgrabber) authenticateClient(cfg IMAPConfig) error {
	imapOps, err := authenticateClient(cfg)
	ig.imapOps = imapOps
	ig.interruptOps = newInterruptOps([]os.Signal{os.Interrupt})
	ig.downloadOps = downloader{
		imapOps:    imapOps,
		deliverOps: deliverer{},
	}
	return err
}

// logout is used to log out from an authenticated session
func (ig *Imapgrabber) logout(doTerminate bool) error {
	defer ig.interruptOps.deregister()
	if doTerminate {
		logInfo("terminating connection")
		return ig.imapOps.Terminate()
	}
	logInfo("logging out")
	return ig.imapOps.Logout()
}

// getFolderList provides all folders in the configured mailbox
func (ig *Imapgrabber) getFolderList() ([]string, error) {
	return getFolderList(ig.imapOps)
}

// downloadMissingEmailsToFolder downloads all emails to a local path that are present remotely
// but missing locally
func (ig *Imapgrabber) downloadMissingEmailsToFolder(
	maildirPath maildirPathT, oldmailName string,
) (err error) {
	if !ig.interruptOps.interrupted() {
		return downloadMissingEmailsToFolder(
			ig.downloadOps, maildirPath, oldmailName, ig.interruptOps,
		)
	}
	return fmt.Errorf("not downloading due to previous interrupt")
}

// NewImapgrabOps creates a new instance of the default implementation of ImapgrabOps.
func NewImapgrabOps() ImapgrabOps {
	return &Imapgrabber{}
}

// GetAllFolders retrieves a list of all folders in a mailbox.
func GetAllFolders(cfg IMAPConfig, ops ImapgrabOps) (folders []string, err error) {
	err = ops.authenticateClient(cfg)
	if err == nil {
		// Make sure to log out in the end if we logged in successfully.
		defer func() {
			// Don't overwrite the error if it has already been set.
			if logoutErr := ops.logout(false); logoutErr != nil && err == nil {
				err = logoutErr
			}
		}()
		// Actually retrieve folder list.
		folders, err = ops.getFolderList()
	}
	return folders, err
}

type threadConfig struct {
	cfg    IMAPConfig
	folder string
}

// func ParallelFolderDownload(
//     cfgs []IMAPConfig, folders []string, maildirBase string, threads int,
// ) (err error) {
//
//     if len(cfgs) != len(folders) {
//         return fmt.Errorf("have %d configs for %d folders", len(cfgs), len(folders))
//     }
//
//     threadCfgs := []threadConfig{}
//     for idx := range cfgs {
//         threadCfgs = append(threadCfgs, threadConfig{cfgs[idx], folders[idx]})
//     }
//
//     // This channel will ensure we never run more than threads download operations at the same time.
//     controller := make(chan bool, threads)
//     // Fill controller up. Each value in the controller means we can spawn one more thread.
//     for idx := 0; idx < threads; idx++ {
//         controller <- true
//     }
//
//     interrupt := newInterruptOps([]os.Signal{os.Interrupt})
//     defer interrupt.register()()
//
//     for threadIdx, threadCfg := range threadCfgs {
//         if interrupted {
//             break
//         }
//         controller <- true
//         go func() {
//             // Fill controller back up.
//             controller <- true
//         }()
//     }
// }

// DownloadFolder downloads all not yet downloaded email from a folder in a mailbox to a maildir.
// The oldmail file in the parent directory of the maildir is used to determine which emails have
// already been downloaded. According to the [maildir specs](https://cr.yp.to/proto/maildir.html),
// the email is first downloaded into the `tmp` sub-directory and then moved atomically to the `new`
// sub-directory.
func DownloadFolder(
	cfg IMAPConfig, folders []string, maildirBase string, ops ImapgrabOps,
) (err error) {
	// Authenticate against the remote server.
	err = ops.authenticateClient(cfg)

	var availableFolders []string
	if err == nil {
		// Make sure to log out in the end if we logged in successfully.
		defer func() {
			if logoutErr := ops.logout(err != nil); logoutErr != nil && err == nil {
				// Don't overwrite the error if it has already been set.
				err = logoutErr
			}
		}()
		// Actually retrieve folder list.
		availableFolders, err = ops.getFolderList()
	}
	if err == nil {
		folders = expandFolders(folders, availableFolders)

		for _, folder := range folders {
			oldmailFilePath := oldmailFileName(cfg, folder)
			maildirPath := maildirPathT{base: maildirBase, folder: folder}

			err = ops.downloadMissingEmailsToFolder(maildirPath, oldmailFilePath)
			if err != nil {
				return err
			}
		}
	}
	return err
}
