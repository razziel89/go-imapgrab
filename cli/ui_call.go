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

type runExeResult struct {
	prettyCmd string
	stdout    string
	stderr    string
}

// Call an executable with arguments and return stdout and stderr. Specify the executable via
// "exe"", the arguments via "args", additional environment variables in the form "key=value" via
// "env", and standard input via "stdin". The command will be cancelled automatically when the
// context expires.
func runExe(
	ctx context.Context, exe string, args []string, env []string, stdin string,
) (runExeResult, error) {
	prettyCmd := fmt.Sprintf("%s %s", exe, strings.Join(quote(args), " "))
	log.Println("Running command:", prettyCmd)

	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Env = env

	cmd.Stdin = strings.NewReader(stdin)

	stdout := strings.Builder{}
	cmd.Stdout = &stdout
	stderr := strings.Builder{}
	cmd.Stderr = &stderr

	err := cmd.Run()

	return runExeResult{
		prettyCmd: prettyCmd,
		stdout:    stdout.String(),
		stderr:    stderr.String(),
	}, err
}

type runExeConf struct {
	exePath string
	args    []string
	stdin   string
	env     []string
	verbose bool
}

// Create a structure that can be used to call a specific command of go-imapgrab. We have to provide
// the path to go-imapgrab since we don't know it and don't want to hardcode it here. This function
// contains very specific knowledge of which commands support which arguments. If the returned error
// is non-nil, then the specified command is now known.
func newRunSelfConf(
	selfPath, cmd string,
	rootConf rootConfigT,
	downloadConf downloadConfigT,
	serveConf serveConfigT,
) (runExeConf, error) {
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
	case "download":
		args = append(args, []string{"--path", downloadConf.path}...)
		for _, folder := range downloadConf.folders {
			args = append(args, []string{"--folder", folder}...)
		}
	case "login":
		// When calling login, the password has to be provided via stdin for now.
		stdin = rootConf.password
	default:
		return runExeConf{}, fmt.Errorf("unknown command %s", cmd)
	}
	env := []string{fmt.Sprintf("%s=%s", passwdEnvVar, rootConf.password)}

	return runExeConf{
		exePath: selfPath,
		args:    args,
		stdin:   stdin,
		env:     env,
		verbose: rootConf.verbose,
	}, nil
}

// Call a specific command of the go-imapgrab executable based on a config.
func runExeAsync(ctx context.Context, cfg runExeConf) func() (string, error) {
	content := []string{}
	var err error

	sync := make(chan bool)
	go func() {
		result, err := runExe(ctx, cfg.exePath, cfg.args, cfg.env, cfg.stdin)

		if err != nil {
			content = append(
				content, fmt.Sprintf("Failure running '%s', logs follow.\n", result.prettyCmd),
			)
		} else {
			content = append(
				content,
				fmt.Sprintf("Success running '%s', logs follow, if any.\n", result.prettyCmd),
			)
		}
		if err == nil && len(result.stdout) != 0 {
			content = append(content, "Normal output:\n")
			content = append(content, result.stdout)
		}
		if (cfg.verbose || err != nil) && len(result.stderr) != 0 {
			content = append(content, "Verbose output:\n")
			content = append(content, result.stderr)
		}
		sync <- true
	}()

	return func() (string, error) {
		<-sync
		return strings.TrimSpace(strings.Join(content, "\n")), err
	}
}
