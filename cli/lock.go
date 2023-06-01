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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rogpeppe/go-internal/lockedfile"
)

const (
	lockfileName = ".go-imapgrab.lock"
	dirPerms     = 0755
)

type lockFn = func(lockfilePath string, timeout time.Duration) (func(), error)

// Since channels can only pass on single types but no tuples, we create this type.
type lockedT struct {
	unlock func()
	err    error
}

func doLock(lockfilePath string) lockedT {
	unlock, err := lockedfile.MutexAt(lockfilePath).Lock()
	return lockedT{
		unlock: unlock,
		err:    err,
	}
}

// Acquire a lock on a lockfile within a specific timeout. Note that this leaks a goroutine if the
// timeout is reached before the lock can be obtained, but there seems to be no way around that.
func lock(lockfilePath string, timeout time.Duration) (func(), error) {
	// Automatically create all elements of the path to the lockflie, if they do not exist.
	err := os.MkdirAll(filepath.Dir(lockfilePath), dirPerms)
	if err != nil {
		return nil, err
	}

	// Obtain lock, honoring the timeout.
	resultChan := make(chan lockedT, 1)
	go func() {
		resultChan <- doLock(lockfilePath)
	}()
	select {
	case <-time.After(timeout):
		return nil, fmt.Errorf("could not acquire lock on %s within %s", lockfilePath, timeout)
	case result := <-resultChan:
		return result.unlock, result.err
	}
}
