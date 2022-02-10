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
	"strings"
	"time"

	"github.com/emersion/go-imap"
)

// Email contains the relevant information about an email.
type Email struct {
	UID       int
	Timestamp time.Time
	// RFC822 is the content of the email according to this RFC.
	RFC822 string

	// Private mebers follow.

	// These members determine which of the fields has already been set. They are used for internal
	// debugging.
	setUID       bool
	setTimestamp bool
	setRFC822    bool
	seenHeader   bool
}

// Function set sets a member of an email depending on the type of the input. It errors out if the
// respective field has already been set.
func (e *Email) set(value interface{}) error {
	switch concrete := value.(type) {
	case uint32:
		if e.setUID {
			return fmt.Errorf("UID already set")
		}
		e.UID = int(concrete)
		e.setUID = true
	case time.Time:
		if e.setTimestamp {
			return fmt.Errorf("timestamp already set")
		}
		e.Timestamp = concrete
		e.setTimestamp = true
	case imap.RawString:
		// Ignore this case. This is a header specification.
	default:
		// Ignore the first entry in this category. It will be the header specification for this
		// RFC. Only throw an error if the string representation of that does not contain RFC822.
		if !e.seenHeader {
			if !strings.Contains(fmt.Sprint(concrete), "RFC822") {
				return fmt.Errorf("RFC822 header not found or with unexpected content")
			}
			e.seenHeader = true
			return nil
		}
		// The second occurrence contains the body.
		if e.setRFC822 {
			return fmt.Errorf("RFC822 already set")
		}
		// This is likely the RFC822 content.
		e.RFC822 = fmt.Sprint(concrete)
		e.setRFC822 = true
	}
	return nil
}

// Function validate returns whether all expected fields of an email have been set.
func (e Email) validate() bool {
	return e.setUID && e.setTimestamp && e.setRFC822
}

// String provides a nice string representation of the email. This contains only the bare content
// and none of the meta data.
func (e Email) String() string {
	return e.RFC822
}
