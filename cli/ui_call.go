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
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// Call an executable with arguments and return stdout and stderr. Specify the command via
// "cmdName"", the arguments via "args", additional environment variables in the form "key=value"
// via "env", and standard input via "stdin".
func callWithArgs(
	ctx context.Context,
	cmdName string,
	args []string,
	env []string,
	stdin string,
) (string, string, error) {
	fullCmd := fmt.Sprintf("%s %s", cmdName, strings.Join(quote(args), " "))
	log.Println("Running command:", fullCmd)

	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Env = env

	cmd.Stdin = strings.NewReader(stdin)

	stdout := strings.Builder{}
	cmd.Stdout = &stdout
	stderr := strings.Builder{}
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("failed to execute '%s': %s", fullCmd, err.Error())
	}

	return stdout.String(), stderr.String(), err
}

type runCmdArgs struct {
	cmd     string
	exe     string
	args    []string
	stdin   string
	env     []string
	verbose bool
}

func runCmdArgsFromConfs(
	cmd, exe string, rootConf rootConfigT, downloadConf downloadConfigT, serveConf serveConfigT,
) (runCmdArgs, error) {
	args := []string{
		cmd,
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
		args = append(args, []string{"--path", downloadConf.path}...)
		for _, folder := range downloadConf.folders {
			args = append(args, []string{"--folder", folder}...)
		}
	case "login":
		// When calling login, the password has to be provided via stdin for now.
		stdin = rootConf.password
	default:
		return runCmdArgs{}, fmt.Errorf("unknown command %s", cmd)
	}
	env := []string{fmt.Sprintf("%s=%s", passwdEnvVar, rootConf.password)}

	return runCmdArgs{
		cmd:     cmd,
		exe:     exe,
		args:    args,
		stdin:   stdin,
		env:     env,
		verbose: rootConf.verbose,
	}, nil
}

// Call a specific command of the go-imapgrab executable based on a root config. We have to provide
// the path to go-imapgrab since we don't know it and don't want to hardcode it here. This function
// contains very specific knowledge of which commands support which arguments.
//
// Make sure the context is cancelled before calling the returned function.
func runFromConfAsync(
	ctx context.Context,
	cfg runCmdArgs,
) func() (string, error) {
	content := []string{}
	var err error

	sync := make(chan bool)
	go func() {
		defer func() { sync <- true }()

		stdout, stderr, err := callWithArgs(ctx, cfg.exe, cfg.args, cfg.env, cfg.stdin)

		if err != nil {
			content = append(
				content, fmt.Sprintf("Failure running '%s', logs follow.\n", cfg.cmd),
			)
		} else {
			content = append(
				content, fmt.Sprintf("Success running '%s', logs follow, if any.\n", cfg.cmd),
			)
		}
		if err == nil && len(stdout) != 0 {
			content = append(content, "Normal output:\n")
			content = append(content, stdout)
		}
		if (cfg.verbose || err != nil) && len(stderr) != 0 {
			content = append(content, "Verbose output:\n")
			content = append(content, stderr)
		}
	}()

	return func() (string, error) {
		<-sync
		return strings.TrimSpace(strings.Join(content, "\n")), err
	}
}
