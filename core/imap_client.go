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

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
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

func selectInbox(imapClient *client.Client, folder string) (*imap.MailboxStatus, error) {
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

func getNthMessage(
	mbox *imap.MailboxStatus, imapClient *client.Client, index int,
) (message *imap.Message, err error) {
	// Make sure there are enough messages in this mailbox and we are not requesting a non-positive
	// index.
	if index <= 0 {
		return nil, fmt.Errorf("message index must be positive")
	}
	emailIdx := int(mbox.Messages) - index + 1
	if emailIdx < 0 {
		err := fmt.Errorf("cannot access %d-th recent email, have only %d", index, mbox.Messages)
		return nil, err
	}

	// Emails will be retrieved via a SeqSet, which can contain a sequential set of messages. Here,
	// we retrieve only one.
	seqset := new(imap.SeqSet)
	seqset.AddRange(uint32(emailIdx), uint32(emailIdx))

	messages := make(chan *imap.Message, 1)
	go func() {
		err = imapClient.Fetch(
			seqset,
			[]imap.FetchItem{
				imap.FetchBody,
				imap.FetchBodyStructure,
				imap.FetchEnvelope,
				imap.FetchFlags,
				imap.FetchInternalDate,
				imap.FetchRFC822,
				imap.FetchRFC822Header,
				imap.FetchRFC822Size,
				imap.FetchRFC822Text,
				imap.FetchUid,
			},
			messages,
		)
	}()
	for m := range messages {
		if message == nil {
			message = m
		} else {
			// Error out if we have somehow retrieved more than one email.
			return nil, fmt.Errorf("internal error, retrieved more than one message")
		}
	}
	return message, err
}
