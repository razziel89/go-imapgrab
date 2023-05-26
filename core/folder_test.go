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
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsGmailDir(t *testing.T) {
	testData := []struct {
		name    string
		isGmail bool
	}{
		{"no-gmail", false},
		{"Inbox", false},
		{"Drafts", false},
		{"Gelöscht", false}, // Deleted in German, with special characters.
		{"", false},         // Empty, should not happen but still no Gmail.
		{"can even have spaces", false},
		{"no-gmail", false},
		{"[Gmail]", true},
		{"[Gmail]/asdf", true},
		{"[Gmail]/anything", true},
		{"[Gmail]/can even have spaces", true},
		{"[Google Mail]/asdf", true},
		{"[Google Mail]/can even have spaces", true},
	}

	for _, data := range testData {
		actual := isGmailDir(data.name)
		if data.isGmail {
			assert.True(t, actual)
		} else {
			assert.False(t, actual)
		}
	}
}

func availableTestFolders() []string {
	folders := []string{
		"folder", "death star", "x-wing", "[Gmail]/emperor", "[Google Mail]/rebels",
	}
	sort.Strings(folders)
	return folders
}

func TestExpandFoldersSelectAll(t *testing.T) {
	selector := []string{"_ALL_"}
	actual := expandFolders(selector, availableTestFolders())
	assert.Equal(t, availableTestFolders(), actual)
}

func TestExpandFoldersDeselectAll(t *testing.T) {
	selector := []string{"_ALL_", "-_ALL_"}
	actual := expandFolders(selector, availableTestFolders())
	assert.Empty(t, actual)
}

func TestExpandFoldersSelectGmail(t *testing.T) {
	selector := []string{"_Gmail_"}
	actual := expandFolders(selector, availableTestFolders())
	assert.Equal(t, []string{"[Gmail]/emperor", "[Google Mail]/rebels"}, actual)
}

func TestExpandFoldersDeselectGmail(t *testing.T) {
	selector := []string{"_ALL_", "-_Gmail_"}
	actual := expandFolders(selector, availableTestFolders())
	assert.Equal(t, []string{"death star", "folder", "x-wing"}, actual)
}

func TestExpandFoldersSelectExistent(t *testing.T) {
	selector := []string{"death star"}
	actual := expandFolders(selector, availableTestFolders())
	assert.Equal(t, []string{"death star"}, actual)
}

func TestExpandFoldersDeselectExistent(t *testing.T) {
	selector := []string{"death star", "-death star"}
	actual := expandFolders(selector, availableTestFolders())
	assert.Empty(t, actual)
}

func TestExpandFoldersSelectNonexistent(t *testing.T) {
	selector := []string{"IDontExist"}
	actual := expandFolders(selector, availableTestFolders())
	assert.Empty(t, actual)
}

func TestExpandFoldersDeselectNonexistent(t *testing.T) {
	selector := []string{"_ALL_", "-IDontExist"}
	actual := expandFolders(selector, availableTestFolders())
	assert.Equal(t, availableTestFolders(), actual)
}

func TestExpandFoldersMultiSelect(t *testing.T) {
	selector := []string{"death star", "death star", "death star"}
	actual := expandFolders(selector, availableTestFolders())
	assert.Equal(t, []string{"death star"}, actual)
}
