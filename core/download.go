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

package core

import (
	"fmt"
	"sync"

	"github.com/emersion/go-imap"
)

const (
	// Buffer for email delivery on disk.
	messageDeliveryBuffer = 10
)

type downloadOps interface {
	selectFolder(folder string) (*imap.MailboxStatus, error)
	getAllMessageUUIDs(*imap.MailboxStatus) ([]uid, error)
	streamingOldmailWriteout(<-chan oldmail, string, *sync.WaitGroup, *sync.WaitGroup) (*int, error)
	streamingRetrieval(
		*imap.MailboxStatus, []rangeT, *sync.WaitGroup, *sync.WaitGroup, func() bool,
	) (<-chan emailOps, *int, error)
	streamingDelivery(
		<-chan emailOps, string, int, *sync.WaitGroup, *sync.WaitGroup,
	) (<-chan oldmail, *int)
}

type downloader struct {
	imapOps    imapOps
	deliverOps deliverOps
}

func (d downloader) selectFolder(folder string) (*imap.MailboxStatus, error) {
	return selectFolder(d.imapOps, folder)
}

func (d downloader) getAllMessageUUIDs(mbox *imap.MailboxStatus) ([]uid, error) {
	return getAllMessageUUIDs(mbox, d.imapOps)
}

func (d downloader) streamingOldmailWriteout(
	deliveredChan <-chan oldmail, oldmailPath string, wg, startWg *sync.WaitGroup,
) (*int, error) {
	return streamingOldmailWriteout(deliveredChan, oldmailPath, wg, startWg)
}

func (d downloader) streamingRetrieval(
	mbox *imap.MailboxStatus,
	missingIDRanges []rangeT,
	wg, startWg *sync.WaitGroup,
	interrupted func() bool,
) (<-chan emailOps, *int, error) {
	return streamingRetrieval(mbox, d.imapOps, missingIDRanges, wg, startWg, interrupted)
}

func (d downloader) streamingDelivery(
	messageChan <-chan emailOps, maildirPath string, uidvalidity int, wg, startWg *sync.WaitGroup,
) (<-chan oldmail, *int) {
	return streamingDelivery(d.deliverOps, messageChan, maildirPath, uidvalidity, wg, startWg)
}

func downloadMissingEmailsToFolder(
	ops downloadOps, maildirPath maildirPathT, oldmailName string, sig interruptOps,
) (err error) {
	oldmails, oldmailPath, err := initMaildir(oldmailName, maildirPath)
	var mbox *imap.MailboxStatus
	if err == nil {
		mbox, err = ops.selectFolder(maildirPath.folderName())
	}
	// Retrieve information about which emails are present on the remote system and check which ones
	// are missing when comparing against those on disk.
	var uidvalidity int
	var uids []uid
	if err == nil && sig.interrupted() {
		err = fmt.Errorf("aborting due to user interrupt")
	}
	if err == nil {
		uidvalidity = int(mbox.UidValidity)
		uids, err = ops.getAllMessageUUIDs(mbox)
	}
	var missingIDRanges []rangeT
	if err == nil {
		missingIDRanges, err = determineMissingIDs(oldmails, uids)
	}
	total := accumulateRanges(missingIDRanges)
	logInfo(fmt.Sprintf("will download %d new emails", total))
	if err != nil || total == 0 {
		return err
	}

	var wg, startWg sync.WaitGroup
	startWg.Add(1) // startWg is used to defer operations until the pipeline is set up.
	// Retrieve email information. This does not download the emails themselves yet.
	messageChan, fetchErrCount, err := ops.streamingRetrieval(
		mbox, missingIDRanges, &wg, &startWg, sig.interrupted,
	)
	var deliveredChan <-chan oldmail
	var deliverErrCount, oldmailErrCount *int
	if err == nil {
		// Download missing emails and store them on disk.
		deliveredChan, deliverErrCount = ops.streamingDelivery(
			messageChan, maildirPath.folderPath(), uidvalidity, &wg, &startWg,
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
			err = fmt.Errorf(msg)
		}
	}
	return err
}
