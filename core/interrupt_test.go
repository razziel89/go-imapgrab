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

func (i *mockInterrupter) register() func() {
	args := i.Called()
	return args.Get(0).(func())
}

func (i *mockInterrupter) deregister() {
	_ = i.Called()
}

func (i *mockInterrupter) interrupt() interruptT {
	args := i.Called()
	return args.Get(0).(interruptT)
}

func (i *mockInterrupter) interrupted() bool {
	args := i.Called()
	return args.Bool(0)
}

func TestInterrupter(t *testing.T) {
	interrupter := newInterruptOps([]os.Signal{os.Interrupt})
	defer interrupter.register()()

	wg := sync.WaitGroup{}
	wg.Add(1)

	receivedSignal := false
	go func() {
		<-interrupter.interrupt()
		receivedSignal = true
		wg.Done()
	}()

	// Sleep a while to be sure we didn't read from the channel yet.
	time.Sleep(time.Millisecond * 100) //nolint:gomnd
	assert.False(t, receivedSignal)

	// Send signal to self.
	self, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)
	err = self.Signal(os.Interrupt)
	assert.NoError(t, err)

	wg.Wait()
	assert.True(t, receivedSignal)
}
