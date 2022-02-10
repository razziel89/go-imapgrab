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
	"strings"

	"github.com/emersion/go-imap"
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

// EmailContent contains the parsed contents of an email in an easily viewable format.
type EmailContent struct {
	Date      string
	Subject   string
	From      []string
	Sender    []string
	ReplyTo   []string
	To        []string
	Cc        []string
	Bcc       []string
	InReplyTo string
	MessageID string
}

func addsToStrings(adds []*imap.Address) []string {
	result := make([]string, 0, len(adds))
	for _, address := range adds {
		converted := address.MailboxName + "@" + address.HostName
		result = append(result, converted)
	}
	return result
}

func envelopeToEmail(env *imap.Envelope) EmailContent {
	return EmailContent{
		Date:      env.Date.String(),
		Subject:   env.Subject,
		From:      addsToStrings(env.From),
		Sender:    addsToStrings(env.Sender),
		ReplyTo:   addsToStrings(env.ReplyTo),
		To:        addsToStrings(env.To),
		Cc:        addsToStrings(env.Cc),
		Bcc:       addsToStrings(env.Bcc),
		InReplyTo: env.InReplyTo,
		MessageID: env.MessageId,
	}
}

// func bodystructureToString(structure *imap.BodyStructure) string {
// 	if structure == nil {
// 		logInfo("cannot convert nil body structure to string")
// 		return ""
// 	}
// 	fields := structure.Format()
// 	strFields := make([]string, 0, len(fields))
// 	for _, field := range fields {
// 		if field != nil {
// 			strFields = append(strFields, fmt.Sprint(field))
// 		}
// 	}
// 	return strings.Join(strFields, ", ")
// }
//
// func literalToString(lit imap.Literal) string {
// 	content, _ := io.ReadAll(lit)
// 	return string(content)
// }
//
// func bodyToString(body map[*imap.BodySectionName]imap.Literal) string {
// 	strFields := make([]string, 0, len(body))
// 	logInfo(fmt.Sprintf("converting %d body fields", len(body)))
// 	for _, lit := range body {
// 		strFields = append(strFields, literalToString(lit))
// 	}
// 	return strings.Join(strFields, ", ")
// }

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

	// body := bodystructureToString(msg.BodyStructure)
	// if len(body) == 0 {
	// 	body = bodyToString(msg.Body)
	// }

	fields := msg.Format()
	strFields := make([]string, 0, len(fields))
	for _, field := range fields {
		strFields = append(strFields, fmt.Sprint(field))
	}
	body := strings.Join(strFields, "\n\n=====================\n\n")

	return fmt.Sprintf("%+v\n%s", envelopeToEmail(msg.Envelope), body), nil
}

// GetAllUIDs obtains all UIDs of all emails in a mailbox.
func GetAllUIDs(cfg IMAPConfig, folder string) (uids []int, err error) {
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
	return getAllMessageUUIDs(mbox, imapClient)
}
