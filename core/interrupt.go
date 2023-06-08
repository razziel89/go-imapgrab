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
	"os/signal"
	"sync"
)

type interruptOps interface {
	deregister()
	interrupted() bool
	wait()
}

type interrupter struct {
	signals []os.Signal
	channel chan os.Signal
	sync.Mutex
}

func (i *interrupter) lock() func() {
	i.Lock()
	return i.Unlock
}

func (i *interrupter) register() func() {
	defer i.lock()()
	signalChan := make(chan os.Signal, len(i.signals))
	signal.Notify(signalChan, i.signals...)
	i.channel = signalChan
	return i.deregister
}

func (i *interrupter) deregisterNoLock() {
	if i.channel != nil {
		signal.Stop(i.channel)
	}
	i.channel = nil
}

func (i *interrupter) deregister() {
	defer i.lock()()
	i.deregisterNoLock()
}

func (i *interrupter) interrupted() bool {
	defer i.lock()()
	if i.channel == nil {
		return true
	}
	select {
	case <-i.channel:
		i.deregisterNoLock()
		return true
	default:
		return false
	}
}

func (i *interrupter) wait() {
	defer i.lock()()
	// Wait first without lock to avoid deadlocks with other methods of this type.
	<-i.channel
	i.deregisterNoLock()
}

func newInterruptOps(signals []os.Signal) interruptOps {
	result := &interrupter{signals: signals}
	result.register()
	return result
}
