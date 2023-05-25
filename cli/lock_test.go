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

package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockAndReleaseSuccess(t *testing.T) {
	tmpdir := t.TempDir()
	lockfile := filepath.Join(tmpdir, "test.lock")

	unlock, err := lock(lockfile, time.Millisecond)
	assert.NoError(t, err)
	unlock()
}

func TestLockCanBeReacquiredAfterRelease(t *testing.T) {
	tmpdir := t.TempDir()
	lockfile := filepath.Join(tmpdir, "test.lock")

	unlock, err := lock(lockfile, time.Millisecond)
	require.NoError(t, err)
	unlock()

	unlock, err = lock(lockfile, time.Millisecond)
	assert.NoError(t, err)
	unlock()
}

func TestLockCannotCreateParentDir(t *testing.T) {
	tmpdir := t.TempDir()
	file := filepath.Join(tmpdir, "not_a_dir")
	// Create an empty file because we cannot create a file under it.
	err := os.WriteFile(file, []byte{}, 0600)
	require.NoError(t, err)
	lockfile := filepath.Join(file, "test.lock")

	_, err = lock(lockfile, time.Millisecond)
	assert.ErrorContains(t, err, "not a directory")
}

func TestLockCannotBeAcquiredMultipleTimes(t *testing.T) {
	tmpdir := t.TempDir()
	lockfile := filepath.Join(tmpdir, "test.lock")
	wg := sync.WaitGroup{}

	var firstLockErr, secondLockErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		unlock, err := lock(lockfile, time.Millisecond)
		firstLockErr = err
		// Due to sleeping, the other goroutine in this test should have enough time to catch up and
		// try to get the lock.
		time.Sleep(50 * time.Millisecond)
		if unlock != nil {
			unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		unlock, err := lock(lockfile, time.Millisecond)
		secondLockErr = err
		// Due to sleeping, the other goroutine in this test should have enough time to catch up and
		// try to get the lock.
		time.Sleep(50 * time.Millisecond)
		if unlock != nil {
			unlock()
		}
	}()

	wg.Wait()
	// Only one of the two goroutines could get the lock. If it's not the first, we simply swap
	// errors to simplify the assertions.
	if firstLockErr != nil && secondLockErr == nil {
		firstLockErr, secondLockErr = secondLockErr, firstLockErr
	}
	assert.NoError(t, firstLockErr)
	assert.ErrorContains(t, secondLockErr, "could not acquire lock")
}
