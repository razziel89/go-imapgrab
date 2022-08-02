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
	"sync"
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
	logInfo(fmt.Sprintf("logging out"))
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

func partitionFolders(folders []string, numPartitions int) ([][]string, int) {
	// Never spawn more threads than there are folders.
	if numPartitions > len(folders) {
		numPartitions = len(folders)
	}

	partitions := make([][]string, numPartitions)

	for idx, folder := range folders {
		partIdx := idx % numPartitions
		partitions[partIdx] = append(partitions[partIdx], folder)
	}

	return partitions, numPartitions
}

// DownloadFolder downloads all not yet downloaded email from a folder in a mailbox to a maildir.
// The oldmail file in the parent directory of the maildir is used to determine which emails have
// already been downloaded. According to the [maildir specs](https://cr.yp.to/proto/maildir.html),
// the email is first downloaded into the `tmp` sub-directory and then moved atomically to the `new`
// sub-directory.
func DownloadFolder(cfg IMAPConfig, folders []string, maildirBase string, threads int) error {
	interrupt := newInterruptOps([]os.Signal{os.Interrupt})
	defer interrupt.deregister()

	errs := threadSafeErrors{verbose: true}

	// The logout happens in each goroutine that uses the ImapgrabOps to download. This one will be
	// used by the first downloading goroutine.
	mainOps := NewImapgrabOps()
	// Authenticate against the remote server.
	errs.add(mainOps.authenticateClient(cfg))

	var availableFolders []string
	if !errs.bad() {
		// Actually retrieve folder list.
		var err error
		availableFolders, err = mainOps.getFolderList()
		errs.add(err)
	}
	if errs.bad() {
		// Special case handling for logout in error case.
		errs.add(mainOps.logout(true))
		return errs.err()
	}
	if threads == 0 {
		threads = len(availableFolders)
	}
	folders = expandFolders(folders, availableFolders)
	partitions, threads := partitionFolders(folders, threads)

	var wg sync.WaitGroup

	for idx := range partitions {
		// We must not close over loop variables in Go. Otherwise, the second iteration will
		// overwrite the partition used in the first iteration's goroutine.
		partition := partitions[idx]
		var ops ImapgrabOps
		if idx > 0 {
			// The first goroutine will use the "ops" we alread have. Every other goroutine will get
			// its own "ops". This way, the mutex implicit in the interrupt handler does not cause
			// deadlocks or slowdowns beocause each gorutine has its own one.
			// After this call, the interrupt signal handler hidden in "ops" will be registered.
			ops = NewImapgrabOps()
			// Authenticate against the remote server.
			errs.add(ops.authenticateClient(cfg))
		} else {
			ops = mainOps
		}
		// The signal handler in "ops" is already registered. Thus, if we have not yet been
		// interrupted, we can be sure the goroutine will receive any interrupt.
		if !interrupt.interrupted() && !errs.bad() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Make sure to log out in the end if we logged in successfully.
				defer func() {
					errs.add(ops.logout(errs.bad()))
				}()
				for _, folder := range partition {
					oldmailFilePath := oldmailFileName(cfg, folder)
					maildirPath := maildirPathT{base: maildirBase, folder: folder}

					downloadErr := ops.downloadMissingEmailsToFolder(maildirPath, oldmailFilePath)
					errs.add(downloadErr)
				}
			}()
		} else {
			errs.add(fmt.Errorf("stopping download threads due to user interrupt or error"))
			break
		}
	}

	// Wait here instead of deferring the wait. Deferred functions are executed after computing the
	// return value, but we want to return the actual error. Thus, wait until all goroutines have
	// finished and return the error value then.
	wg.Wait()
	return errs.err()
}
