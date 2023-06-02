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
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockFile struct {
	m       *mock.Mock
	content string
}

func (f *mockFile) Write(b []byte) (n int, err error) {
	args := f.m.Called(b)
	return args.Int(0), args.Error(1)
}

func (f *mockFile) Close() error {
	args := f.m.Called()
	return args.Error(0)
}

func (f *mockFile) Read(p []byte) (n int, err error) {
	if len(f.content) > len(p) {
		panic("buffer too short")
	}
	copy(p, f.content)
	args := f.m.Called(p)
	return args.Int(0), args.Error(1)
}

func setUpMockOldmailFile() (mockFile, func()) {
	orgOpenFile := openFile

	fileMock := mockFile{&mock.Mock{}, ""}

	openFile = func(_ string, _ int, _ fs.FileMode) (fileOps, error) {
		return &fileMock, nil
	}

	deferMe := func() {
		openFile = orgOpenFile
	}

	return fileMock, deferMe
}

func TestOldmailToString(t *testing.T) {
	om := oldmail{
		uidFolder: 100,
		uid:       42,
		timestamp: 12345,
	}
	// Unix 12345 is equal to: Thu  1 Jan 03:25:45 UTC 1970
	expectedOldmailFormat := "100/42 -> 1970-01-01 03:25:45 +0000 UTC"

	assert.Equal(t, expectedOldmailFormat, om.String())
}

func TestOldmailFileName(t *testing.T) {
	name := oldmailFileName(
		IMAPConfig{
			Server:   "some_server",
			Port:     42,
			User:     "some_user",
			Password: "not contained in file name",
		},
		"some_folder",
	)

	assert.Equal(t, "oldmail-some_server-42-some_user-some_folder", name)
}

func TestOldmailRead(t *testing.T) {
	expectedOldmails := []oldmail{
		{uidFolder: 123, uid: 21, timestamp: 747},
		{uidFolder: 123, uid: 42, timestamp: 447},
		{uidFolder: 123, uid: 11, timestamp: 321},
		{uidFolder: 123, uid: 15, timestamp: 898},
		{uidFolder: 123, uid: 17, timestamp: 242},
	}
	oldmailContent := []byte(
		"123/21_747\n" +
			"123/42_447\n" +
			"123/11_321\n" +
			"123/15_898\n" +
			"123/17_242\n",
	)
	// Replace "_" by the null character to work around go strings ignoring the null byte.
	oldmailContent = bytes.ReplaceAll(oldmailContent, []byte("_"), []byte{0})

	tmp := t.TempDir()
	tmpFile := filepath.Join(tmp, "tmpfile")

	err := os.WriteFile(tmpFile, oldmailContent, 0400)
	assert.NoError(t, err)

	oldmails, err := readOldmail(tmpFile)
	assert.NoError(t, err)

	assert.Equal(t, expectedOldmails, oldmails)
}

func TestOldmailReadFileNotFound(t *testing.T) {
	tmp := t.TempDir()
	tmpFile := filepath.Join(tmp, "tmpfile")

	_, err := readOldmail(tmpFile)

	assert.Error(t, err)
}

func TestOldmailReadCloseError(t *testing.T) {
	f, cleanUp := setUpMockOldmailFile()
	defer cleanUp()

	tmpFile := filepath.Join(t.TempDir(), "tmpfile")
	err := touch(tmpFile, 0444)
	assert.NoError(t, err)

	f.m.On("Close").Return(fmt.Errorf("some error"))
	f.m.On("Read", mock.Anything).Return(0, io.EOF)

	_, err = readOldmail(tmpFile)

	assert.Error(t, err)
	f.m.AssertExpectations(t)
}

func TestOldmailReadReadError(t *testing.T) {
	f, cleanUp := setUpMockOldmailFile()
	defer cleanUp()

	tmpFile := filepath.Join(t.TempDir(), "tmpfile")
	err := touch(tmpFile, 0444)
	assert.NoError(t, err)

	f.m.On("Close").Return(nil)
	f.m.On("Read", mock.Anything).Return(0, fmt.Errorf("some error"))

	_, err = readOldmail(tmpFile)

	assert.Error(t, err)
	f.m.AssertExpectations(t)
}

func TestOldmailReadParseError(t *testing.T) {
	f, cleanUp := setUpMockOldmailFile()
	defer cleanUp()

	f.content = "too-few-fields"

	tmpFile := filepath.Join(t.TempDir(), "tmpfile")
	err := touch(tmpFile, 0444)
	assert.NoError(t, err)

	f.m.On("Close").Return(nil)
	f.m.On("Read", mock.Anything).Return(len(f.content), io.EOF)

	_, err = readOldmail(tmpFile)

	assert.Error(t, err)
	f.m.AssertExpectations(t)
}

func TestOldmailReadCannotRead(t *testing.T) {
	tmp := t.TempDir()
	tmpFile := filepath.Join(tmp, "tmpfile")

	err := os.WriteFile(tmpFile, []byte{}, 0400)
	assert.NoError(t, err)

	err = os.Chmod(tmpFile, 0000)
	assert.NoError(t, err)

	_, err = readOldmail(tmpFile)

	assert.Error(t, err)
}

func TestOldmailWriteout(t *testing.T) {
	oldmails := []oldmail{
		{uidFolder: 123, uid: 21, timestamp: 747},
		{uidFolder: 123, uid: 42, timestamp: 447},
		{uidFolder: 123, uid: 11, timestamp: 321},
		{uidFolder: 123, uid: 15, timestamp: 898},
		{uidFolder: 123, uid: 17, timestamp: 242},
	}
	expectedOldmailContent := []byte(
		"123/21_747\n" +
			"123/42_447\n" +
			"123/11_321\n" +
			"123/15_898\n" +
			"123/17_242\n",
	)
	// Replace "_" by the null character to work around go strings ignoring the null byte.
	expectedOldmailContent = bytes.ReplaceAll(expectedOldmailContent, []byte("_"), []byte{0})

	oldmailChan := make(chan oldmail)
	go func() {
		for _, om := range oldmails {
			oldmailChan <- om
		}
		close(oldmailChan)
	}()

	tmp := t.TempDir()
	tmpFile := filepath.Join(tmp, "tmpfile")

	var wg, stwg sync.WaitGroup
	stwg.Add(1)

	errCountPtr, err := streamingOldmailWriteout(oldmailChan, tmpFile, &wg, &stwg)
	assert.NoError(t, err)

	// Nothing written before stwg has been set to done.
	time.Sleep(time.Millisecond * 100)
	content, err := os.ReadFile(tmpFile) // nolint: gosec
	assert.NoError(t, err)
	assert.Empty(t, content)

	// Stuff is written after stwg was set to done.
	stwg.Done()
	wg.Wait()
	content, err = os.ReadFile(tmpFile) // nolint: gosec
	assert.NoError(t, err)
	assert.Equal(t, expectedOldmailContent, content)
	assert.Zero(t, *errCountPtr)
}

func TestOldmailWriteoutCannotWriteToFile(t *testing.T) {
	tmp := t.TempDir()
	tmpPath := filepath.Join(tmp, "tmpfile")

	// Trigger an error below by having a directory where a file was expected.
	err := os.Mkdir(tmpPath, 0500)
	assert.NoError(t, err)

	oldmailChan := make(chan oldmail)
	close(oldmailChan)
	var wg, stwg sync.WaitGroup

	_, err = streamingOldmailWriteout(oldmailChan, tmpPath, &wg, &stwg)
	assert.Error(t, err)
}

func TestOldmailWriteoutWriteAndCloseError(t *testing.T) {
	f, cleanUp := setUpMockOldmailFile()
	defer cleanUp()

	f.m.On("Write", mock.Anything).Return(0, fmt.Errorf("some write error"))
	f.m.On("Close").Return(fmt.Errorf("some close error"))

	var wg, stwg sync.WaitGroup

	oldmailChan := make(chan oldmail)
	go func() {
		oldmailChan <- oldmail{}
		oldmailChan <- oldmail{}
		close(oldmailChan)
	}()

	errCountPtr, err := streamingOldmailWriteout(oldmailChan, "not-needed", &wg, &stwg)
	assert.NoError(t, err)

	wg.Wait()

	assert.Equal(t, 2, *errCountPtr)
	f.m.AssertExpectations(t)

	// Ensure that was the last value in the channel.
	_, ok := <-oldmailChan
	assert.False(t, ok)
}
