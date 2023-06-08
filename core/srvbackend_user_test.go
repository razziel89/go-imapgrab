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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackendUserDisallowWriteOperations(t *testing.T) {
	user := igrabUser{}

	err := user.CreateMailbox("")
	assert.ErrorIs(t, err, errReadOnlyServer)

	err = user.DeleteMailbox("")
	assert.ErrorIs(t, err, errReadOnlyServer)

	err = user.RenameMailbox("", "")
	assert.ErrorIs(t, err, errReadOnlyServer)

	// User has not been changed.
	assert.Equal(t, igrabUser{}, user)
}

func TestBackendUserUsernameLogout(t *testing.T) {
	user := igrabUser{name: "someone"}

	name := user.Username()
	assert.Equal(t, "someone", name)

	err := user.Logout()
	assert.NoError(t, err)
}

func TestBackendUserGetListMailbox(t *testing.T) {
	box := igrabMailbox{maildir: maildirPathT{base: "base", folder: "folder"}}
	user := igrabUser{
		mailboxes: []*igrabMailbox{&box},
	}

	boxes, err := user.ListMailboxes(true)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(boxes))
	assert.Equal(t, &box, boxes[0].(*igrabMailbox))

	gotten, err := user.GetMailbox("folder")

	assert.NoError(t, err)
	assert.Equal(t, &box, gotten.(*igrabMailbox))

	_, err = user.GetMailbox("unknown")

	assert.Error(t, err)
}

func TestBackendAddMailboxes(t *testing.T) {
	verbose = true
	tmp := filepath.Join(t.TempDir(), "base")
	_, _, err := initMaildir("oldmail", maildirPathT{base: tmp, folder: "folder"})

	assert.NoError(t, err)

	user := igrabUser{}
	err = user.addMailboxes(tmp)
	assert.NoError(t, err)
}

func TestBackendAddMailboxesMissingDir(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "base")
	user := igrabUser{}
	err := user.addMailboxes(tmp)
	assert.Error(t, err)
}
