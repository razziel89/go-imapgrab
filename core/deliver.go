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

import "sync"

type deliverOps interface {
	deliverMessage(string, string) error
	rfc822FromEmail(emailOps, uidFolder) (string, oldmail, error)
}

type deliverer struct{}

func (d deliverer) deliverMessage(text string, maildirPath string) error {
	return deliverMessage(text, maildirPath)
}

func (d deliverer) rfc822FromEmail(msg emailOps, uidFolder uidFolder) (string, oldmail, error) {
	return rfc822FromEmail(msg, uidFolder)
}

func streamingDelivery(
	ops deliverOps,
	messageChan <-chan emailOps,
	maildirPath string,
	uidFolder uidFolder,
	wg, stwg *sync.WaitGroup,
) (returnedChan <-chan oldmail, errCountPtr *int) {
	var errCount int

	deliveredChan := make(chan oldmail, messageDeliveryBuffer)

	wg.Add(1)
	go func() {
		// Do not start before the entire pipeline has been set up.
		stwg.Wait()
		for msg := range messageChan {
			// Deliver each email to the `tmp` directory and move them to the `new` directory.
			text, oldmail, err := ops.rfc822FromEmail(msg, uidFolder)
			if err == nil {
				err = ops.deliverMessage(text, maildirPath)
			}
			if err != nil {
				logError(err.Error())
				errCount++
				continue
			}
			deliveredChan <- oldmail
		}
		wg.Done()
		close(deliveredChan)
	}()

	return deliveredChan, &errCount
}
