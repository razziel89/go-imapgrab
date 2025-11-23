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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockEmail struct {
	uid int
	mock.Mock
}

func (e *mockEmail) Format() []interface{} {
	args := e.Called()
	return args.Get(0).([]interface{})
}

func TestEmailSetValidateStringSuccess(t *testing.T) {
	e := email{}

	for _, val := range []interface{}{
		uint32(1),
		time.Now(),
		string("some header"),
		"rfc822 header",
		"actual content",
	} {
		err := e.set(val)
		assert.NoError(t, err)
	}

	assert.True(t, e.validate())
	assert.Equal(t, "actual content", e.String())
}

func TestEmailSetValueTwice(t *testing.T) {
	for _, val := range []interface{}{
		uint32(1),
		time.Now(),
	} {
		t.Log("setting", val)
		e := email{}
		err := e.set(val)
		assert.NoError(t, err)
		err = e.set(val)
		assert.Error(t, err)
	}
}

func TestEmailSetRFCTooOften(t *testing.T) {
	e := email{}
	err := e.set("rfc822 header")
	assert.NoError(t, err)
	err = e.set("content")
	assert.NoError(t, err)
	err = e.set("too many strings")
	assert.Error(t, err)
}

func TestEmailSetNoRFCHeader(t *testing.T) {
	e := email{}
	err := e.set("the first string needs the rfc header")
	assert.Error(t, err)
}

func TestRFCFromEmail(t *testing.T) {
	someTime := time.Now()
	someTimestamp := int(someTime.UTC().Unix())
	msg := mockEmail{}
	// Yes, this is actually what this function returns. You first get a header and then the actual
	// content. I have not found another way to extract the rfc822 content because there is no real
	// way to find out which header/content pair comes at what position in the slice.
	msg.On("Format").Return(
		[]interface{}{
			string("uid header"),
			uint32(1),
			string("time header"),
			someTime,
			"rfc822 header",
			"actual content",
		},
	)

	content, om, err := rfc822FromEmail(&msg, 21)
	assert.NoError(t, err)
	assert.Equal(t, "actual content", content)
	assert.Equal(t, oldmail{uidFolder: 21, uid: 1, timestamp: someTimestamp}, om)
	msg.AssertExpectations(t)
}

func TestRFCFromEmailTooFewFields(t *testing.T) {
	msg := mockEmail{}
	msg.On("Format").Return(
		[]interface{}{
			string("uid header"),
			uint32(1),
			string("time header"),
			time.Now(),
			// No content.
		},
	)

	_, _, err := rfc822FromEmail(&msg, 21)
	assert.Error(t, err)
	msg.AssertExpectations(t)
}

func TestRFCFromEmailEnoughFieldsButUnexpectedType(t *testing.T) {
	msg := mockEmail{}
	msg.On("Format").Return(
		[]interface{}{
			42, // We never expect an int anywhere in the slice.
			42,
			42,
			42,
			42,
			42,
		},
	)

	_, _, err := rfc822FromEmail(&msg, 21)
	assert.Error(t, err)
	msg.AssertExpectations(t)
}

func TestRFCFromEmailEnoughFieldsButNotAllWeNeed(t *testing.T) {
	msg := mockEmail{}
	msg.On("Format").Return(
		[]interface{}{
			// We expect 6 entries, but we ignore all headers.
			string("header"),
			string("header"),
			string("header"),
			string("header"),
			string("header"),
			string("header"),
		},
	)

	_, _, err := rfc822FromEmail(&msg, 21)
	assert.Error(t, err)
	msg.AssertExpectations(t)
}
