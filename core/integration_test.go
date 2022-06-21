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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func buildFakeImapMessage(t *testing.T, id uint32, content string) *imap.Message {
	sectionName, err := imap.ParseBodySectionName(imap.FetchItem("RFC822"))
	assert.NoError(t, err)

	buf := bytes.NewBufferString(content)

	return &imap.Message{
		Uid: id,
		Items: map[imap.FetchItem]interface{}{
			"INTERNALDATE": nil,
			"RFC822":       nil,
			"UID":          nil,
		},
		Body: map[*imap.BodySectionName]imap.Literal{
			sectionName: buf,
		},
	}
}

func buildFakeDownloader(imapOps imapOps) *downloader {
	return &downloader{
		imapOps:    imapOps,
		deliverOps: deliverer{},
	}
}

func TestIntegrationDownloadMissingEmailsToFolderSuccess(t *testing.T) {
	if val, found := os.LookupEnv("SKIP_INTEGRATION_TESTS"); !found || val != "0" {
		t.Skip("integration tests disabled")
	}

	orgVerbosity := verbose
	SetVerboseLogs(true)
	t.Cleanup(func() { SetVerboseLogs(orgVerbosity) })

	mockPath := setUpEmptyMaildir(t, "some-folder", "some-oldmail")

	boxes := []*imap.MailboxInfo{&imap.MailboxInfo{Name: "some-folder"}}
	status := &imap.MailboxStatus{Name: "some-folder", UidValidity: 42, Messages: 3}
	messages := []*imap.Message{
		buildFakeImapMessage(t, 1, "some text"),
		buildFakeImapMessage(t, 2, "some more text"),
		buildFakeImapMessage(t, 3, "even more text"),
	}

	seqSet := &imap.SeqSet{}
	seqSet.AddRange(1, 3)
	fetchRequestListUUIDs := []imap.FetchItem{imap.FetchUid, imap.FetchInternalDate}
	fetchRequestDownload := []imap.FetchItem{
		imap.FetchUid, imap.FetchInternalDate, imap.FetchRFC822,
	}

	mockClient := setUpMockClient(t, boxes, messages, nil)
	mockClient.On("Select", "some-folder", true).Return(status, nil)
	mockClient.On("Fetch", seqSet, fetchRequestListUUIDs, mock.Anything).Return(nil)
	mockClient.On("Fetch", seqSet, fetchRequestDownload, mock.Anything).Return(nil)

	maildirPath := maildirPathT{base: mockPath, folder: "some-folder"}

	downloader := buildFakeDownloader(mockClient)
	interrupter := newInterruptOps(nil)

	err := downloadMissingEmailsToFolder(downloader, maildirPath, "some-oldmail", interrupter)

	assert.NoError(t, err)

	// Check whether emails have actually been downloaded and whether hte oldmail file has been
	// updated.
	oldmailContent, err := ioutil.ReadFile(filepath.Join(mockPath, "some-oldmail")) // nolint: gosec
	assert.NoError(t, err)
	downloadedMessages, err := ioutil.ReadDir(filepath.Join(mockPath, "some-folder", "new"))
	assert.NoError(t, err)
	// Oldmail file contains three lines.
	assert.Equal(t, 3, bytes.Count(oldmailContent, []byte("\n")))
	// New directory contains three files.
	assert.Equal(t, 3, len(downloadedMessages))
}

func TestIntegrationDownloadMissingEmailsToFolderPreparationError(t *testing.T) {
	if val, found := os.LookupEnv("SKIP_INTEGRATION_TESTS"); !found || val != "0" {
		t.Skip("integration tests disabled")
	}

	orgVerbosity := verbose
	SetVerboseLogs(true)
	t.Cleanup(func() { SetVerboseLogs(orgVerbosity) })

	mockPath := setUpEmptyMaildir(t, "some-folder", "some-oldmail")

	boxes := []*imap.MailboxInfo{&imap.MailboxInfo{Name: "some-folder"}}
	status := &imap.MailboxStatus{Name: "some-folder", UidValidity: 42, Messages: 0}
	// No emails, thus nothing to be downloaded.
	messages := []*imap.Message{}

	mockClient := setUpMockClient(t, boxes, messages, nil)
	mockClient.On("Select", "some-folder", true).Return(status, fmt.Errorf("some error"))

	maildirPath := maildirPathT{base: mockPath, folder: "some-folder"}

	downloader := buildFakeDownloader(mockClient)
	interrupter := newInterruptOps(nil)

	err := downloadMissingEmailsToFolder(downloader, maildirPath, "some-oldmail", interrupter)

	assert.Error(t, err)
	assert.Equal(t, "some error", err.Error())
}

func TestIntegrationDownloadMissingEmailsToFolderDownloadError(t *testing.T) {
	if val, found := os.LookupEnv("SKIP_INTEGRATION_TESTS"); !found || val != "0" {
		t.Skip("integration tests disabled")
	}

	orgVerbosity := verbose
	SetVerboseLogs(true)
	t.Cleanup(func() { SetVerboseLogs(orgVerbosity) })

	mockPath := setUpEmptyMaildir(t, "some-folder", "some-oldmail")

	boxes := []*imap.MailboxInfo{&imap.MailboxInfo{Name: "some-folder"}}
	status := &imap.MailboxStatus{Name: "some-folder", UidValidity: 42, Messages: 3}
	messages := []*imap.Message{
		buildFakeImapMessage(t, 1, "some text"),
		buildFakeImapMessage(t, 2, "some more text"),
		// One of the messages does not contain the information we need, which will cause an error
		// in the streaming email delivery that will be logged.
		&imap.Message{},
	}

	seqSet := &imap.SeqSet{}
	seqSet.AddRange(1, 3)
	fetchRequestListUUIDs := []imap.FetchItem{imap.FetchUid, imap.FetchInternalDate}
	fetchRequestDownload := []imap.FetchItem{
		imap.FetchUid, imap.FetchInternalDate, imap.FetchRFC822,
	}

	mockClient := setUpMockClient(t, boxes, messages, nil)
	mockClient.On("Select", "some-folder", true).Return(status, nil)
	mockClient.On("Fetch", seqSet, fetchRequestListUUIDs, mock.Anything).Return(nil)

	// Cause an error when retrieving emails because one email cannot be downloaded. Every
	// successive download succeeds.
	mockClient.On("Fetch", seqSet, fetchRequestDownload, mock.Anything).
		Once().Return(fmt.Errorf("download error"))

	maildirPath := maildirPathT{base: mockPath, folder: "some-folder"}

	downloader := buildFakeDownloader(mockClient)
	interrupter := newInterruptOps(nil)

	err := downloadMissingEmailsToFolder(downloader, maildirPath, "some-oldmail", interrupter)

	assert.Error(t, err)
	assert.Equal(
		t, "there were 1/1/0 errors while: retrieving/delivering/remembering mail", err.Error(),
	)

	// Check whether we could still download all successfully that were delivered and whether that
	// email's information has been added to the oldmail file.
	oldmailContent, err := ioutil.ReadFile(filepath.Join(mockPath, "some-oldmail")) // nolint: gosec
	assert.NoError(t, err)
	downloadedMessages, err := ioutil.ReadDir(filepath.Join(mockPath, "some-folder", "new"))
	assert.NoError(t, err)
	// Oldmail file contains two lines.
	assert.Equal(t, 2, bytes.Count(oldmailContent, []byte("\n")))
	// New directory contains two files.
	assert.Equal(t, 2, len(downloadedMessages))
}
