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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThreadSafeErrorsLifeCycle(t *testing.T) {
	errs := threadSafeErrors{}
	assert.NoError(t, errs.err())
	assert.False(t, errs.bad())

	// Adding no error will not cause the state to turn bad.
	errs.add(nil)
	assert.NoError(t, errs.err())
	assert.False(t, errs.bad())

	// Adding an error will cause the state to turn bad.
	errs.add(fmt.Errorf("some error"))
	assert.Error(t, errs.err())
	assert.True(t, errs.bad())
	assert.Contains(t, errs.err().Error(), "some error")

	// Adding no error will keep the state bad.
	errs.add(nil)
	assert.Error(t, errs.err())
	assert.True(t, errs.bad())
}
