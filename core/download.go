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
	"sync"

	"github.com/emersion/go-imap/v2"
)

const (
	// Buffer for email delivery on disk.
	messageDeliveryBuffer = 10
)

type downloadOps interface {
	selectFolder(folder string) (*imap.SelectData, error)
	getAllMessageUUIDs(*imap.SelectData) ([]uidExt, error)
	streamingOldmailWriteout(<-chan oldmail, string, *sync.WaitGroup, *sync.WaitGroup) (*int, error)
	streamingRetrieval(
		[]uid, *sync.WaitGroup, *sync.WaitGroup, func() bool,
	) (<-chan emailOps, *int, error)
	streamingDelivery(
		<-chan emailOps, string, uidFolder, *sync.WaitGroup, *sync.WaitGroup,
	) (<-chan oldmail, *int)
}

type downloader struct {
	imapOps    imapOps
	deliverOps deliverOps
}

func (d downloader) selectFolder(folder string) (*imap.SelectData, error) {
	return selectFolder(d.imapOps, folder)
}

func (d downloader) getAllMessageUUIDs(mbox *imap.SelectData) ([]uidExt, error) {
	return getAllMessageUUIDs(mbox, d.imapOps)
}

func (d downloader) streamingOldmailWriteout(
	deliveredChan <-chan oldmail, oldmailPath string, wg, startWg *sync.WaitGroup,
) (*int, error) {
	return streamingOldmailWriteout(deliveredChan, oldmailPath, wg, startWg)
}

func (d downloader) streamingRetrieval(
	missingUIDs []uid,
	wg, startWg *sync.WaitGroup,
	interrupted func() bool,
) (<-chan emailOps, *int, error) {
	return streamingRetrieval(d.imapOps, missingUIDs, wg, startWg, interrupted)
}

func (d downloader) streamingDelivery(
	messageChan <-chan emailOps,
	maildirPath string,
	uidFolder uidFolder,
	wg, startWg *sync.WaitGroup,
) (<-chan oldmail, *int) {
	return streamingDelivery(d.deliverOps, messageChan, maildirPath, uidFolder, wg, startWg)
}

func downloadMissingEmailsToFolder(
	ops downloadOps, maildirPath maildirPathT, oldmailName string, sig interruptOps,
) (err error) {
	oldmails, oldmailPath, err := initMaildir(oldmailName, maildirPath)
	var mbox *imap.SelectData
	if err == nil {
		mbox, err = ops.selectFolder(maildirPath.folderName())
	}
	// Retrieve information about which emails are present on the remote system and check which ones
	// are missing when comparing against those on disk.
	var uidFold uidFolder
	var uids []uidExt
	if err == nil && sig.interrupted() {
		err = fmt.Errorf("aborting due to user interrupt")
	}
	if err == nil {
		uidFold = uidFolder(mbox.UIDValidity)
		uids, err = ops.getAllMessageUUIDs(mbox)
	}
	var missingUIDs []uid
	if err == nil {
		missingUIDs, err = determineMissingUIDs(oldmails, uids)
	}
	total := len(missingUIDs)
	logInfo(fmt.Sprintf("will download %d new emails", total))
	if err != nil || total == 0 {
		return err
	}

	var wg, startWg sync.WaitGroup
	startWg.Add(1) // startWg is used to defer operations until the pipeline is set up.
	// Retrieve email information. This does not download the emails themselves yet.
	messageChan, fetchErrCount, err := ops.streamingRetrieval(
		missingUIDs, &wg, &startWg, sig.interrupted,
	)
	var deliveredChan <-chan oldmail
	var deliverErrCount, oldmailErrCount *int
	if err == nil {
		// Download missing emails and store them on disk.
		deliveredChan, deliverErrCount = ops.streamingDelivery(
			messageChan, maildirPath.folderPath(), uidFold, &wg, &startWg,
		)
		// Retrieve and write out information about all emails.
		oldmailErrCount, err = ops.streamingOldmailWriteout(
			deliveredChan, oldmailPath, &wg, &startWg,
		)
	}
	if err == nil {
		// Wait until all has been processed and report on errors.
		startWg.Done()
		wg.Wait()
		msg := fmt.Sprintf(
			"there were %d/%d/%d errors while: retrieving/delivering/remembering mail",
			*fetchErrCount, *deliverErrCount, *oldmailErrCount,
		)
		logInfo(msg)
		if *fetchErrCount > 0 || *deliverErrCount > 0 || *oldmailErrCount > 0 {
			err = fmt.Errorf("%s", msg)
		}
	}
	return err
}
