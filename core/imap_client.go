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
	"io"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

const (
	folderListBuffer       = 10
	messageRetrievalBuffer = 20
)

// Make this a function pointer to simplify testing. Also takes a boolean to decide whether to use
// secure auth nor not (i.e. TLS). This errors out if insecure auth is chossen but anything other
// than "127.0.0.1" is passed as "addr".
var newImapClient = func(addr string, insecure bool) (imapOps, error) {
	var client *imapclient.Client
	var err error
	
	if !insecure {
		// Use automatic configuration of TLS options.
		client, err = imapclient.DialTLS(addr, nil)
	} else if !strings.HasPrefix(addr, "127.0.0.1:") {
		err = fmt.Errorf(
			"not allowing insecure auth for non-localhost address %s, use 127.0.0.1", addr,
		)
	} else {
		logWarning("using insecure connection to locahost")
		client, err = imapclient.DialInsecure(addr, nil)
	}
	
	if err != nil {
		return nil, err
	}
	
	return client, nil
}

// imapOps defines the IMAP operations interface. In v2, this is implemented by *imapclient.Client.
type imapOps interface {
	Login(username, password string) *imapclient.Command
	List(ref, name string, options *imap.ListOptions) *imapclient.ListCommand
	Select(name string, options *imap.SelectOptions) *imapclient.SelectCommand
	Fetch(numSet imap.NumSet, options *imap.FetchOptions) *imapclient.FetchCommand
	Logout() *imapclient.Command
	Close() error
}

// messageWrapper wraps v2 message data to implement the Format() interface expected by email.go
type messageWrapper struct {
	uid          uint32
	internalDate time.Time
	body         []byte
}

func (m *messageWrapper) Format() []interface{} {
	var fields []interface{}
	// Add UID header and value
	fields = append(fields, "UID")
	fields = append(fields, m.uid)
	// Add INTERNALDATE header and value
	fields = append(fields, "INTERNALDATE")
	fields = append(fields, m.internalDate)
	// Add RFC822 header and body
	fields = append(fields, "RFC822")
	fields = append(fields, string(m.body))
	return fields
}

func authenticateClient(config IMAPConfig) (imapClient imapOps, err error) {
	if len(config.Password) == 0 {
		logError("empty password detected")
		err = fmt.Errorf("password not set")
		return nil, err
	}

	logInfo(fmt.Sprintf("connecting to server %s", config.Server))
	serverWithPort := fmt.Sprintf("%s:%d", config.Server, config.Port)
	if imapClient, err = newImapClient(serverWithPort, config.Insecure); err != nil {
		logError("cannot connect")
		return nil, err
	}
	logInfo("connected")

	logInfo(fmt.Sprintf("logging in as %s with provided password", config.User))
	if err = imapClient.Login(config.User, config.Password).Wait(); err != nil {
		logError("cannot log in")
		return nil, err
	}
	logInfo("logged in")

	return imapClient, nil
}

func getFolderList(imapClient imapOps) (folders []string, err error) {
	logInfo("retrieving folders")
	
	listCmd := imapClient.List("", "*", nil)
	defer listCmd.Close()
	
	for {
		mbox := listCmd.Next()
		if mbox == nil {
			break
		}
		folders = append(folders, mbox.Mailbox)
	}
	
	if err = listCmd.Close(); err != nil {
		return nil, err
	}
	
	logInfo(fmt.Sprintf("retrieved %d folders", len(folders)))
	return folders, nil
}

func selectFolder(imapClient imapOps, folder string) (*imap.SelectData, error) {
	logInfo(fmt.Sprint("selecting folder:", folder))
	// Access the folder in read-only mode.
	opts := &imap.SelectOptions{ReadOnly: true}
	mbox, err := imapClient.Select(folder, opts).Wait()
	if err == nil {
		logInfo(fmt.Sprint("flags for selected folder are", mbox.Flags))
		logInfo(fmt.Sprintf("selected folder contains %d emails", mbox.NumMessages))
	}
	return mbox, err
}

// Type once behaves like sync.Once but we can also query whether it has already been called. This
// is needed because sync.Once does not provide a facility to check that.
type once struct {
	called bool
	hook   func()
	sync.Once
}

func (o *once) call() {
	o.Do(o.hook)
}

func newOnce(hook func()) *once {
	o := once{}
	innerHook := func() {
		o.called = true
		hook()
	}
	o.hook = innerHook
	return &o
}

// Obtain messages whose ids/indices lie in certain ranges. Negative indices are automatically
// converted to count from the last message. That is, -1 refers to the most recent message while 1
// refers to the second oldest email.
//
// In this function, we translate from *imap.Message to emailOps separately. Sadly, the compiler
// does not auto-generate the code to use a `chan emailOps` as a `chan *imap.Message`. Thus, we need
// a separate, second goroutine translating between the two. This second goroutine also handles
// interrupts.
func streamingRetrieval(
	imapClient imapOps,
	uids []uid,
	wg, startWg *sync.WaitGroup,
	interrupted func() bool,
) (returnedChan <-chan emailOps, errCountPtr *int, err error) {
	// Make sure all UIDs are >0.
	for _, uid := range uids {
		if uid <= 0 {
			return nil, nil, fmt.Errorf("detected a UID<=0, aborting")
		}
	}

	// Emails will be retrieved via a UIDSet in v2
	uidSet := imap.UIDSet{}
	for _, u := range uids {
		uidSet.AddNum(imap.UID(u))
	}

	wg.Add(1)
	// Ensure we call "Done" exactly once on wg here.
	already := newOnce(func() { wg.Done() })
	var errCount int
	translatedMessageChan := make(chan emailOps, messageRetrievalBuffer)
	
	go func() {
		defer close(translatedMessageChan)
		// Do not start before the entire pipeline has been set up.
		startWg.Wait()
		
		// Set up fetch options for UID, InternalDate, and RFC822 body
		fetchOpts := &imap.FetchOptions{
			UID:          true,
			InternalDate: true,
			BodySection: []*imap.FetchItemBodySection{
				{Specifier: imap.PartSpecifierNone}, // Fetch entire message (RFC822)
			},
		}
		
		fetchCmd := imapClient.Fetch(uidSet, fetchOpts)
		defer fetchCmd.Close()
		
		for {
			if interrupted() {
				errCount++
				already.call()
				logWarning("caught keyboard interrupt, closing connection")
				break
			}
			
			msg := fetchCmd.Next()
			if msg == nil {
				break
			}
			
			// Collect message data
			wrapper := &messageWrapper{}
			
			for {
				item := msg.Next()
				if item == nil {
					break
				}
				
				switch item := item.(type) {
				case imapclient.FetchItemDataUID:
					wrapper.uid = uint32(item.UID)
				case imapclient.FetchItemDataInternalDate:
					wrapper.internalDate = item.Time
				case imapclient.FetchItemDataBodySection:
					if item.Literal != nil {
						body, err := io.ReadAll(item.Literal)
						if err != nil {
							logError(fmt.Sprintf("error reading body: %v", err))
							errCount++
						} else {
							wrapper.body = body
						}
					}
				}
			}
			
			translatedMessageChan <- wrapper
		}
		
		if err := fetchCmd.Close(); err != nil {
			logError(err.Error())
			errCount++
		}
		already.call()
	}()

	return translatedMessageChan, &errCount, nil
}

// Type uid describes a message. It is a type alias to prevent accidental mixups.
type uid int

// Type uidFolder describes a mailbox. It is a type alias to prevent accidental mixups.
type uidFolder int

// Type uidExt describes a unique identifier for a message as well as the associated mailbox. It
// consists of the unique identifier of the mailbox the message belongs to and a unique identifier
// for a message within that mailbox.
type uidExt struct {
	folder uidFolder
	msg    uid
}

// String provides a string representation for a message's unique identifier.
func (u uidExt) String() string {
	return fmt.Sprintf("%d/%d", u.folder, u.msg)
}

func getAllMessageUUIDs(
	mbox *imap.SelectData, imapClient imapOps,
) (uids []uidExt, err error) {
	logInfo("retrieving information about emails stored on server")
	// Handle the special case of empty folders by returning early.
	if mbox.NumMessages == 0 {
		return nil, nil
	}

	uids = make([]uidExt, 0, mbox.NumMessages)

	// Retrieve information about all emails using sequence numbers
	seqSet := imap.SeqSet{}
	seqSet.AddRange(1, mbox.NumMessages)

	// Set up fetch options for UID and InternalDate
	fetchOpts := &imap.FetchOptions{
		UID:          true,
		InternalDate: true,
	}

	fetchCmd := imapClient.Fetch(seqSet, fetchOpts)
	defer fetchCmd.Close()

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		var msgUID imap.UID
		for {
			item := msg.Next()
			if item == nil {
				break
			}

			if uidItem, ok := item.(imapclient.FetchItemDataUID); ok {
				msgUID = uidItem.UID
			}
		}

		if msgUID > 0 {
			appUID := uidExt{
				folder: uidFolder(mbox.UIDValidity),
				msg:    uid(msgUID),
			}
			uids = append(uids, appUID)
		}
	}

	if err = fetchCmd.Close(); err != nil {
		return nil, err
	}

	logInfo(fmt.Sprintf("received information for %d emails", len(uids)))
	return uids, nil
}
