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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setUpEmptyMaildir(t *testing.T, folderName, oldmailName string) string {
	// Create simulated empty maildir with empty oldmail file and all the required directories.
	tmpdir := t.TempDir()
	for _, dir := range []string{"cur", "new", "tmp"} {
		err := os.MkdirAll(filepath.Join(tmpdir, folderName, dir), dirPerm)
		assert.NoError(t, err)
	}
	err := os.WriteFile(filepath.Join(tmpdir, oldmailName), []byte{}, filePerm)
	assert.NoError(t, err)
	return tmpdir
}

func TestNewUniqueNameSuccess(t *testing.T) {
	// Use a set to ensure each name is really unique.
	set := map[string]struct{}{}
	hostname, err := os.Hostname()
	assert.NoError(t, err)

	for i := 0; i < 100; i++ {
		currentDeliveryCount := deliveryCount.get()

		newName, err := newUniqueName("")

		assert.NoError(t, err)
		assert.Greater(t, deliveryCount.get(), currentDeliveryCount)
		assert.NotContains(t, set, newName)
		assert.Contains(t, newName, hostname)

		set[newName] = struct{}{}
	}
}

func TestNewUniqueNameFailure(t *testing.T) {
	currentDeliveryCount := deliveryCount.get()

	_, err := newUniqueName("hostname with space breaks function")

	assert.Error(t, err)
	assert.Greater(t, deliveryCount.get(), currentDeliveryCount)
}

func TestNewUniqueNameBrokenNameFixes(t *testing.T) {
	currentDeliveryCount := deliveryCount.get()

	newName, err := newUniqueName("BrokenHostname/withSlash")

	assert.NoError(t, err)
	assert.Greater(t, deliveryCount.get(), currentDeliveryCount)
	assert.Contains(t, newName, "BrokenHostname\\057withSlash")
}

func TestNewUniqueNameStartAndEnd(t *testing.T) {
	newName, err := newUniqueName("SomeHost")

	assert.NoError(t, err)
	// The following regex means:
	// - start with at least one digit followed by a dot
	// - end with dot followed by hostname
	// - contain the middle string with some information, see newUniqueName for details what the
	//   individual bits mean
	assert.Regexp(t, "^[0-9]+\\.M[0-9]+P[0-9]+Q[0-9]+R[a-fA-F0-9]+\\.SomeHost$", newName)
}

func TestIsMaildirSuccess(t *testing.T) {
	tmpdir := setUpEmptyMaildir(t, "folder", "oldmail")

	check := isMaildir(filepath.Join(tmpdir, "folder"))

	assert.True(t, check)
}

func TestIsMaildirFailure(t *testing.T) {
	tmpdir := t.TempDir()

	// An empty directory does not contain any of the directories that make a maildir.
	check := isMaildir(tmpdir)

	assert.False(t, check)
}

func TestInitExistingMaildirSuccess(t *testing.T) {
	tmpdir := setUpEmptyMaildir(t, "folder", "oldmail")
	pathVals := maildirPathT{base: tmpdir, folder: "folder"}

	oldmails, oldmailFilePath, err := initExistingMaildir("oldmail", pathVals)

	assert.NoError(t, err)
	assert.Empty(t, oldmails)
	assert.Equal(t, filepath.Join(tmpdir, "oldmail"), oldmailFilePath)
}

func TestInitExistingMaildirMissingSubdir(t *testing.T) {
	tmpdir := setUpEmptyMaildir(t, "folder", "oldmail")
	pathVals := maildirPathT{base: tmpdir, folder: "folder"}

	// If one of the required directories is missing, the target is no maildir.
	err := os.RemoveAll(filepath.Join(tmpdir, "folder", "cur"))
	assert.NoError(t, err)

	_, _, err = initExistingMaildir("oldmail", pathVals)

	assert.Error(t, err)
}

func TestInitExistingMaildirErrorReadingOldmail(t *testing.T) {
	tmpdir := setUpEmptyMaildir(t, "folder", "oldmail")
	pathVals := maildirPathT{base: tmpdir, folder: "folder"}

	// A missing oldmail file counts as an error.
	err := os.Remove(filepath.Join(tmpdir, "oldmail"))
	assert.NoError(t, err)

	_, _, err = initExistingMaildir("oldmail", pathVals)

	assert.Error(t, err)
}

func TestInitMaildirSuccess(t *testing.T) {
	tmpdir := t.TempDir()
	pathVals := maildirPathT{base: tmpdir, folder: "folder"}

	oldmails, oldmailFilePath, err := initMaildir("oldmail", pathVals)

	assert.NoError(t, err)
	assert.Empty(t, oldmails)
	assert.Equal(t, filepath.Join(tmpdir, "oldmail"), oldmailFilePath)

	// Use the function tested above to assert that we did indeed create a maildir. This makes sure
	// there is consistency within the code base.
	assert.True(t, isMaildir(filepath.Join(tmpdir, "folder")))
	assert.FileExists(t, filepath.Join(tmpdir, "oldmail"))
}

func TestInitMaildirFailure(t *testing.T) {
	// Create a file where a directory would be created to cause an error.
	tmpdir := setUpEmptyMaildir(t, "fake_file_actually_dir", "fake_dir_actually_file")
	pathVals := maildirPathT{base: tmpdir, folder: "fake_dir_actually_file"}

	_, _, err := initMaildir("oldmail", pathVals)

	assert.Error(t, err)
}

func TestDeliverMessage(t *testing.T) {
	tmpdir := setUpEmptyMaildir(t, "folder", "oldmail")
	basepath := filepath.Join(tmpdir, "folder")

	err := deliverMessage("I am some text", basepath)

	assert.NoError(t, err)

	// Check that the correct content was written to the only file in the "new" directory.
	files, err := os.ReadDir(filepath.Join(basepath, "new"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(files))

	content, err := os.ReadFile(filepath.Join(basepath, "new", files[0].Name())) // nolint: gosec
	assert.NoError(t, err)
	assert.Equal(t, string(content), "I am some text")

	// The other directories "cur" and "tmp" need to be empty.
	// cur
	files, err = os.ReadDir(filepath.Join(basepath, "cur"))
	assert.NoError(t, err)
	assert.Empty(t, files)
	// tmp
	files, err = os.ReadDir(filepath.Join(basepath, "tmp"))
	assert.NoError(t, err)
	assert.Empty(t, files)
}
