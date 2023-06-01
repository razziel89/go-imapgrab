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
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setUpLogTest() (*bytes.Buffer, func()) {
	buf := bytes.Buffer{}
	log.SetOutput(&buf)

	deferMe := func() {
		log.SetOutput(os.Stderr)
	}

	return &buf, deferMe
}

func TestLogInfoNoVerbose(t *testing.T) {
	buf, cleanUp := setUpLogTest()
	defer cleanUp()

	// Expect nothing to be logged at info level without high verbosity.
	SetVerboseLogs(false)
	logInfo("some message")
	assert.Equal(t, "", buf.String())
}

func TestLogInfoVerbose(t *testing.T) {
	buf, cleanUp := setUpLogTest()
	defer cleanUp()

	// Expect something to be logged at info level with high verbosity.
	SetVerboseLogs(true)
	logInfo("some message")
	assert.Contains(t, buf.String(), "INFO some message")
}

func TestLogWarningNoVerbose(t *testing.T) {
	buf, cleanUp := setUpLogTest()
	defer cleanUp()

	SetVerboseLogs(false)
	logWarning("some message")
	assert.Contains(t, buf.String(), "WARNING some message")
}

func TestLogWarningVerbose(t *testing.T) {
	buf, cleanUp := setUpLogTest()
	defer cleanUp()

	SetVerboseLogs(true)
	logWarning("some message")
	assert.Contains(t, buf.String(), "WARNING some message")
}

func TestLogErrorNoVerbose(t *testing.T) {
	buf, cleanUp := setUpLogTest()
	defer cleanUp()

	SetVerboseLogs(false)
	logError("some message")
	assert.Contains(t, buf.String(), "ERROR some message")
}

func TestLogErrorVerbose(t *testing.T) {
	buf, cleanUp := setUpLogTest()
	defer cleanUp()

	SetVerboseLogs(true)
	logError("some message")
	assert.Contains(t, buf.String(), "ERROR some message")
}
