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
	"log"
)

const logJoiner = ", "

var verbose = false

// SetVerboseLogs sets the log level for core functionality to verbose if passed true and to less
// verbose if passed false.
func SetVerboseLogs(verb bool) {
	verbose = verb
}

func logInfo(msg string) {
	if verbose {
		log.Println("INFO", msg)
	}
}

func logWarning(msg string) {
	// Always log warning.
	log.Println("WARNING", msg)
}

func logError(msg string) {
	// Always log errors.
	log.Println("ERROR", msg)
}
