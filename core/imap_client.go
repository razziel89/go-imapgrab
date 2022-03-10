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
	"github.com/emersion/go-imap/client"
)

const (
	folderListBuffer       = 10
	messageRetrievalBuffer = 20
)

func authenticateClient(config IMAPConfig) (imapClient *client.Client, err error) {
	if len(config.Password) == 0 {
		logError("empty password detected")
		err = fmt.Errorf("password not set")
		return nil, err
	}

	logInfo(fmt.Sprintf("connecting to server %s", config.Server))
	serverWithPort := fmt.Sprintf("%s:%d", config.Server, config.Port)
	if imapClient, err = client.DialTLS(serverWithPort, nil); err != nil {
		logError("cannot connect")
		return nil, err
	}
	logInfo("connected")

	logInfo(fmt.Sprintf("logging in as %s with provided password", config.User))
	if err = imapClient.Login(config.User, config.Password); err != nil {
		logError("cannot log in")
		return nil, err
	}
	logInfo("logged in")

	return imapClient, nil
}

func getFolderList(imapClient *client.Client) (folders []string, err error) {
	logInfo("retrieving folders")
	mailboxes := make(chan *imap.MailboxInfo, folderListBuffer)
	go func() {
		err = imapClient.List("", "*", mailboxes)
	}()
	for m := range mailboxes {
		folders = append(folders, m.Name)
	}
	logInfo(fmt.Sprintf("retrieved %d folders", len(folders)))

	return folders, err
}

func selectFolder(imapClient *client.Client, folder string) (*imap.MailboxStatus, error) {
	logInfo(fmt.Sprint("selecting folder:", folder))
	// Access the folder in read-only mode.
	mbox, err := imapClient.Select(folder, true)
	if err != nil {
		return nil, err
	}
	logInfo(fmt.Sprint("flags for selected folder are", mbox.Flags))
	logInfo(fmt.Sprintf("selected folder contains %d emails", mbox.Messages))
	return mbox, err
}

// Obtain messages whose ids/indices lie in certain ranges. Negative indices are automatically
// converted to count from the last message. That is, -1 refers to the most recent message while 1
// refers to the second oldest email.
func streamingRetrieval(
	mbox *imap.MailboxStatus, imapClient *client.Client, indices []rangeT, wg *sync.WaitGroup,
) (returnedChan <-chan *imap.Message, errCountPtr *int, err error) {
	// Make sure there are enough messages in this mailbox and we are not requesting a non-positive
	// index.
	indices, err = canonicalizeRanges(indices, 1, int(mbox.Messages)+1)
	if err != nil {
		return nil, nil, err
	}

	// Emails will be retrieved via a SeqSet, which can contain a set of messages.
	seqset := new(imap.SeqSet)
	for _, r := range indices {
		seqset.AddRange(uint32(r.start), uint32(r.end-1))
	}

	wg.Add(1)
	var errCount int
	messageChan := make(chan *imap.Message, messageRetrievalBuffer)
	go func() {
		err := imapClient.Fetch(
			seqset,
			[]imap.FetchItem{imap.FetchUid, imap.FetchInternalDate, imap.FetchRFC822},
			messageChan,
		)
		if err != nil {
			logError(err.Error())
			errCount++
		}
		wg.Done()
	}()

	return messageChan, &errCount, nil
}

// Type uid describes a unique identifier for a message. It consists of the unique identifier of the
// mailbox the message belongs to and a unique identifier for a message within that mailbox.
type uid struct {
	Mbox    int
	Message int
}

// String provides a string representation for a message's unique identifier.
func (u uid) String() string {
	return fmt.Sprintf("%d/%d", u.Mbox, u.Message)
}

func getAllMessageUUIDs(
	mbox *imap.MailboxStatus, imapClient *client.Client,
) (uids []uid, err error) {

	logInfo("retrieving information about emails stored on server")
	uids = make([]uid, 0, mbox.Messages)

	// Retrieve information about all emails.
	seqset := new(imap.SeqSet)
	seqset.AddRange(1, mbox.Messages)

	messageChannel := make(chan *imap.Message, messageRetrievalBuffer)
	go func() {
		err = imapClient.Fetch(
			seqset,
			[]imap.FetchItem{imap.FetchUid, imap.FetchInternalDate},
			messageChannel,
		)
	}()
	for m := range messageChannel {
		if m != nil {
			appUID := uid{
				Mbox:    int(mbox.UidValidity),
				Message: int(m.Uid),
			}
			uids = append(uids, appUID)
		}
	}
	logInfo(fmt.Sprintf("received information for %d emails", len(uids)))

	return uids, err
}
