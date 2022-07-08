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
	"os/signal"
	"sync"
)

type interruptT <-chan os.Signal

type interruptOps interface {
	register() func()
	deregister()
	interrupt() interruptT
	interrupted() bool
	uninterruptible(func()) bool
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

func (i *interrupter) interrupt() interruptT {
	defer i.lock()()
	return i.channel
}

func (i *interrupter) uninterruptible(fn func()) bool {
	defer i.lock()()
	if i.channel == nil {
		return false
	}
	select {
	case <-i.channel:
		i.deregisterNoLock()
		return false
	default:
		fn()
		return true
	}
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

func newInterruptOps(signals []os.Signal) interruptOps {
	result := &interrupter{signals: signals}
	result.register()
	return result
}
