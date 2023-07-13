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
	"log"
	"os/exec"
	"strings"
)

// Call an executable with arguments and return stdout and stderr. Specify the command via
// "cmdName"", the arguments via "args", additional environment variables in the form "key=value"
// via "env", and standard input via "stdin".
//
// TODO: allow putting the command in the background and killing it later via some means.
func callWithArgs(
	cmdName string,
	args []string,
	env []string,
	stdin string,
) (string, string, error) {
	log.Println("Running command:", cmdName, strings.Join(quote(args), " "))

	cmd := exec.Command(cmdName, args...)
	cmd.Env = env

	cmd.Stdin = strings.NewReader(stdin)

	stdout := strings.Builder{}
	cmd.Stdout = &stdout
	stderr := strings.Builder{}
	cmd.Stderr = &stderr

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}
