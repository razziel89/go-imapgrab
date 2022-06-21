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
	"os"
	"os/signal"
)

type interruptT <-chan os.Signal

type interruptOps interface {
	register() func()
	deregister()
	interrupt() interruptT
}

type interrupter struct {
	signals []os.Signal
	channel chan os.Signal
}

func (i *interrupter) register() func() {
	signalChan := make(chan os.Signal, len(i.signals))
	signal.Notify(signalChan, i.signals...)
	i.channel = signalChan
	return i.deregister
}

func (i *interrupter) deregister() {
	signal.Stop(i.channel)
	i.channel = nil
}

func (i *interrupter) interrupt() interruptT {
	return i.channel
}

func newInterruptOps(signals []os.Signal) interruptOps {
	return &interrupter{signals: signals}
}

func recoverFromPanic(fn func(string), outerErr *error) {
	recovered := recover()
	if recovered == nil {
		return
	}

	var err error
	var ok bool
	if err, ok = recovered.(error); !ok {
		err = fmt.Errorf("%v", recovered)
	}
	fn(err.Error())
	if outerErr != nil && *outerErr == nil {
		*outerErr = err
	}
}
