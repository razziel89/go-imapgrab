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

// Call a specific command of the go-imapgrab executable based on a root config. We have to provide
// the path to go-imapgrab since we don't know it and don't want to hardcode it here. This function
// contains very specific knowledge of which commands support which arguments.
func runFromConf(
	exe, cmd string,
	rootConf rootConfigT,
	downloadConf downloadConfigT,
	serveConf serveConfigT,
) (string, error) {
	content := []string{}

	// Construct equivalent CLI arguments.
	args := []string{
		// Always ignore keyring, we are using env vars instead to pass the password.
		"--no-keyring",
		"--server", rootConf.server,
		"--user", rootConf.username,
		"--port", fmt.Sprint(rootConf.port),
	}
	if rootConf.verbose {
		args = append(args, "--verbose")
	}
	stdin := ""
	switch cmd {
	case "list":
		// No additional arguments have to be specified.
	case "serve":
		args = append(args, []string{"--server-port", fmt.Sprint(serveConf.serverPort)}...)
		args = append(args, []string{"--path", serveConf.path}...)
		log.Fatal("cannot yet serve, don't know how to shut down", args)
	case "download":
		for _, folder := range downloadConf.folders {
			args = append(args, []string{"--folder", folder}...)
		}
		args = append(args, []string{"--path", downloadConf.path}...)
	case "login":
		// When calling login, the password has to be provided via stdin for now.
		stdin = rootConf.password
	default:
		return "", fmt.Errorf("unknown command %s", cmd)
	}

	stdout, stderr, err := callWithArgs(
		exe,
		args,
		[]string{fmt.Sprintf("%s=%s", passwdEnvVar, rootConf.password)},
		stdin,
	)

	if err != nil {
		content = append(
			content, fmt.Sprintf("Failure, errors follow.\n"),
		)
		content = append(content, err.Error())
	} else {
		content = append(content, fmt.Sprintf("Success, logs follow.\n"))
	}
	if len(stdout) != 0 {
		content = append(content, fmt.Sprintf("Stdout:\n"))
		content = append(content, stdout)
	}
	if len(stderr) != 0 {
		content = append(content, fmt.Sprintf("Stderr:\n"))
		content = append(content, stderr)
	}

	if err == nil {
		return strings.Join(content, "\n"), nil
	}
	return "", fmt.Errorf(strings.Join(content, "\n"))
}
