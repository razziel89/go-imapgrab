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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendNew(t *testing.T) {
	tmp := t.TempDir()
	bcknd, err := newBackend(tmp, "username", "password")
	assert.NoError(t, err)

	assert.Empty(t, bcknd.(*serverBackend).user.mailboxes)
}

func TestBackendLogin(t *testing.T) {
	tmp := t.TempDir()
	bcknd, err := newBackend(tmp, "username", "password")
	require.NoError(t, err)

	_, err = bcknd.Login(nil, "bad user", "password")
	assert.Error(t, err)

	_, err = bcknd.Login(nil, "username", "bad password")
	assert.Error(t, err)

	user, err := bcknd.Login(nil, "username", "password")
	assert.NoError(t, err)
	assert.Equal(t, "username", user.Username())
}
