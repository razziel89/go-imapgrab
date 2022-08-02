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
	"sync"
)

type threadSafeErrors struct {
	errs    []string
	verbose bool
	sync.Mutex
}

func (t *threadSafeErrors) add(err error) {
	if err != nil {
		if t.verbose {
			logError(err.Error())
		}
		t.Lock()
		defer t.Unlock()
		t.errs = append(t.errs, err.Error())
	}
}

func (t *threadSafeErrors) bad() bool {
	t.Lock()
	defer t.Unlock()
	return len(t.errs) != 0
}

func (t *threadSafeErrors) err() error {
	t.Lock()
	defer t.Unlock()
	if len(t.errs) == 0 {
		return nil
	}
	return fmt.Errorf("%d errors detected: %s", len(t.errs), strings.Join(t.errs, ", "))
}

type threadSafeCounter struct {
	count int
	sync.Mutex
}

func (t *threadSafeCounter) inc() int {
	t.Lock()
	defer t.Unlock()
	t.count++
	return t.count
}
