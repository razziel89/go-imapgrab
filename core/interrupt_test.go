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
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockInterrupter struct {
	mock.Mock
}

func (i *mockInterrupter) deregister() {
	_ = i.Called()
}

func (i *mockInterrupter) interrupted() bool {
	args := i.Called()
	return args.Bool(0)
}

func (i *mockInterrupter) wait() {
	_ = i.Called()
}

func TestInterrupter(t *testing.T) {
	interrupter := newInterruptOps([]os.Signal{os.Interrupt})
	defer interrupter.deregister()

	wg := sync.WaitGroup{}
	wg.Add(1)

	receivedSignal := false
	go func() {
		// We check whether we have been interrupted again and again just like what would be done
		// before downloading each email. Ensure we try for longer than we wait further down.
		maxTries := 1000 //nolint:gomnd
		for try := 0; try <= maxTries; try++ {
			if interrupter.interrupted() {
				break
			}
			time.Sleep(time.Millisecond) //nolint:gomnd
		}
		receivedSignal = true
		wg.Done()
	}()

	// Sleep a while to be sure the above loop already went through a few iterations.
	time.Sleep(time.Millisecond * 100) //nolint:gomnd
	assert.False(t, receivedSignal)
	assert.False(t, interrupter.interrupted())

	// Send signal to self.
	self, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)
	err = self.Signal(os.Interrupt)
	assert.NoError(t, err)

	wg.Wait()
	assert.True(t, receivedSignal)
	assert.True(t, interrupter.interrupted())
}

func TestInterrupterNoChannel(t *testing.T) {
	interrupter := interrupter{}
	assert.True(t, interrupter.interrupted())
}

func TestInterrupterWait(t *testing.T) {
	interrupter := newInterruptOps([]os.Signal{os.Interrupt})
	defer interrupter.deregister()

	wg := sync.WaitGroup{}
	wg.Add(1)

	waited := false
	go func() {
		interrupter.wait()
		waited = true
		wg.Done()
	}()

	// Sleep a while to be sure the above goroutine got to the point where it is waiting.
	time.Sleep(time.Millisecond * 100) //nolint:gomnd
	assert.False(t, waited)
	assert.False(t, interrupter.interrupted())

	// Send signal to self.
	self, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)
	err = self.Signal(os.Interrupt)
	assert.NoError(t, err)

	wg.Wait()
	assert.True(t, waited)
	assert.True(t, interrupter.interrupted())
}
