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

// v2ClientAdapter wraps the v2 Client to implement the imapOps interface
type v2ClientAdapter struct {
	client *imapclient.Client
}

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
	
	return &v2ClientAdapter{client: client}, nil
}

func (a *v2ClientAdapter) Login(username, password string) error {
	return a.client.Login(username, password).Wait()
}

func (a *v2ClientAdapter) List(ref, name string, ch chan *v1MailboxInfo) error {
	defer close(ch)
	
	listCmd := a.client.List(ref, name, nil)
	for {
		mbox := listCmd.Next()
		if mbox == nil {
			break
		}
		ch <- &v1MailboxInfo{Name: mbox.Mailbox}
	}
	
	return listCmd.Close()
}

func (a *v2ClientAdapter) Select(name string, readOnly bool) (*v1MailboxStatus, error) {
	opts := &imap.SelectOptions{ReadOnly: readOnly}
	data, err := a.client.Select(name, opts).Wait()
	if err != nil {
		return nil, err
	}
	
	return &v1MailboxStatus{
		Flags:       data.Flags,
		Messages:    data.NumMessages,
		UidValidity: uint32(data.UIDValidity),
	}, nil
}

func (a *v2ClientAdapter) Fetch(seqset *v1SeqSet, items []v1FetchItem, ch chan *v1Message) error {
	return a.doFetch(false, seqset, items, ch)
}

func (a *v2ClientAdapter) UidFetch(seqset *v1SeqSet, items []v1FetchItem, ch chan *v1Message) error {
	return a.doFetch(true, seqset, items, ch)
}

func (a *v2ClientAdapter) doFetch(useUID bool, seqset *v1SeqSet, items []v1FetchItem, ch chan *v1Message) error {
	defer close(ch)
	
	// Convert v1 sequence set to v2 NumSet
	var numSet imap.NumSet
	if useUID {
		uidSet := imap.UIDSet{}
		for _, num := range seqset.nums {
			uidSet.AddNum(imap.UID(num))
		}
		numSet = uidSet
	} else {
		seqSetV2 := imap.SeqSet{}
		for _, num := range seqset.nums {
			seqSetV2.AddNum(num)
		}
		numSet = seqSetV2
	}
	
	// Convert v1 fetch items to v2 options
	opts := &imap.FetchOptions{}
	var needRFC822 bool
	for _, item := range items {
		switch item {
		case v1FetchUid:
			opts.UID = true
		case v1FetchInternalDate:
			opts.InternalDate = true
		case v1FetchRFC822:
			needRFC822 = true
			// RFC822 requires fetching the entire body
			opts.BodySection = []*imap.FetchItemBodySection{
				{Specifier: imap.PartSpecifierNone},
			}
		}
	}
	
	fetchCmd := a.client.Fetch(numSet, opts)
	defer fetchCmd.Close()
	
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}
		
		v1Msg := &v1Message{
			Body: make(map[string][]byte),
		}
		
		for {
			item := msg.Next()
			if item == nil {
				break
			}
			
			switch item := item.(type) {
			case imapclient.FetchItemDataUID:
				v1Msg.Uid = uint32(item.UID)
			case imapclient.FetchItemDataInternalDate:
				v1Msg.InternalDate = item.Time
			case imapclient.FetchItemDataBodySection:
				if needRFC822 && item.Section.Specifier == imap.PartSpecifierNone {
					// Read the body
					if item.Literal != nil {
						body, err := io.ReadAll(item.Literal)
						if err != nil {
							logError(fmt.Sprintf("error reading body: %v", err))
							continue
						}
						v1Msg.Body["RFC822"] = body
					}
				}
			}
		}
		
		ch <- v1Msg
	}
	
	return fetchCmd.Close()
}

func (a *v2ClientAdapter) Logout() error {
	err := a.client.Logout().Wait()
	a.client.Close()
	return err
}

func (a *v2ClientAdapter) Terminate() error {
	return a.client.Close()
}

// v1MailboxInfo is a compatibility type for v1's MailboxInfo
type v1MailboxInfo struct {
	Name string
}

// v1MailboxStatus is a compatibility type for v1's MailboxStatus
type v1MailboxStatus struct {
	Flags        []imap.Flag
	Messages     uint32
	UidValidity  uint32
}

// v1SeqSet is a compatibility type for v1's SeqSet
type v1SeqSet struct {
	nums []uint32
}

func newV1SeqSet() *v1SeqSet {
	return &v1SeqSet{nums: make([]uint32, 0)}
}

func (s *v1SeqSet) AddNum(num uint32) {
	s.nums = append(s.nums, num)
}

func (s *v1SeqSet) AddRange(start, stop uint32) {
	for i := start; i <= stop; i++ {
		s.nums = append(s.nums, i)
	}
}

// v1Message is a compatibility type for v1's Message
type v1Message struct {
	Uid          uint32
	InternalDate time.Time
	Body         map[string][]byte
}

func (m *v1Message) Format() []interface{} {
	var fields []interface{}
	// Add UID header and value (to match v1 format)
	fields = append(fields, "UID")
	fields = append(fields, m.Uid)
	// Add INTERNALDATE header and value
	fields = append(fields, "INTERNALDATE")
	fields = append(fields, m.InternalDate)
	
	// Add RFC822 body if present
	if body, ok := m.Body["RFC822"]; ok {
		fields = append(fields, "RFC822")
		fields = append(fields, string(body))
		// Format returns 6 fields total: "UID", uid_value, "INTERNALDATE", date_value, "RFC822", body_content
		// This matches the rfc822ExpectedNumFields constant.
	}
	
	return fields
}

type v1FetchItem string

const (
	v1FetchUid          v1FetchItem = "UID"
	v1FetchInternalDate v1FetchItem = "INTERNALDATE"
	v1FetchRFC822       v1FetchItem = "RFC822"
)

type imapOps interface {
	Login(username string, password string) error
	List(ref string, name string, ch chan *v1MailboxInfo) error
	Select(name string, readOnly bool) (*v1MailboxStatus, error)
	Fetch(seqset *v1SeqSet, items []v1FetchItem, ch chan *v1Message) error
	UidFetch(seqset *v1SeqSet, items []v1FetchItem, ch chan *v1Message) error
	Logout() error
	Terminate() error
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
	if err = imapClient.Login(config.User, config.Password); err != nil {
		logError("cannot log in")
		return nil, err
	}
	logInfo("logged in")

	return imapClient, nil
}

func getFolderList(imapClient imapOps) (folders []string, err error) {
	logInfo("retrieving folders")
	mailboxes := make(chan *v1MailboxInfo, folderListBuffer)
	go func() {
		err = imapClient.List("", "*", mailboxes)
	}()
	for m := range mailboxes {
		folders = append(folders, m.Name)
	}
	logInfo(fmt.Sprintf("retrieved %d folders", len(folders)))

	return folders, err
}

func selectFolder(imapClient imapOps, folder string) (*v1MailboxStatus, error) {
	logInfo(fmt.Sprint("selecting folder:", folder))
	// Access the folder in read-only mode.
	mbox, err := imapClient.Select(folder, true)
	if err == nil {
		logInfo(fmt.Sprint("flags for selected folder are", mbox.Flags))
		logInfo(fmt.Sprintf("selected folder contains %d emails", mbox.Messages))
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

	// Emails will be retrieved via a SeqSet, which can contain a set of messages.
	seqset := newV1SeqSet()
	for _, uid := range uids {
		seqset.AddNum(intToUint32(int(uid)))
	}

	wg.Add(1)
	// Ensure we call "Done" exactly once on wg here.
	already := newOnce(func() { wg.Done() })
	var errCount int
	translatedMessageChan := make(chan emailOps, messageRetrievalBuffer)
	orgMessageChan := make(chan *v1Message)
	go func() {
		// Do not start before the entire pipeline has been set up.
		startWg.Wait()
		err := imapClient.UidFetch(
			seqset,
			[]v1FetchItem{v1FetchUid, v1FetchInternalDate, v1FetchRFC822},
			orgMessageChan,
		)
		if err != nil {
			logError(err.Error())
			errCount++
		}
		already.call()
	}()

	go func() {
		defer close(translatedMessageChan)
		for !already.called {
			if interrupted() {
				errCount++
				already.call()
				logWarning("caught keyboard interrupt, closing connection")
				// Clean up and report.
			} else {
				msg := <-orgMessageChan
				// Ignore nil values that we sometimes receive even though we should not.
				if msg != nil {
					// Here, the compiler generates code to convert `*imap.Message` into emailOps`.
					translatedMessageChan <- msg
				}
			}
		}
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
	mbox *v1MailboxStatus, imapClient imapOps,
) (uids []uidExt, err error) {
	logInfo("retrieving information about emails stored on server")
	// Handle the special case of empty folders by returning early.
	if mbox.Messages == 0 {
		return nil, nil
	}

	uids = make([]uidExt, 0, mbox.Messages)

	// Retrieve information about all emails.
	seqset := newV1SeqSet()
	seqset.AddRange(1, mbox.Messages)

	messageChannel := make(chan *v1Message, messageRetrievalBuffer)
	go func() {
		err = imapClient.Fetch(
			seqset,
			[]v1FetchItem{v1FetchUid, v1FetchInternalDate},
			messageChannel,
		)
	}()
	for m := range messageChannel {
		if m != nil {
			appUID := uidExt{
				folder: uidFolder(mbox.UidValidity),
				msg:    uid(m.Uid),
			}
			uids = append(uids, appUID)
		}
	}
	logInfo(fmt.Sprintf("received information for %d emails", len(uids)))

	return uids, err
}
