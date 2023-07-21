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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunExeSuccess(t *testing.T) {
	ctx := context.Background()

	result, err := runExe(ctx, "cat", []string{"-"}, nil, "This is input!")

	assert.NoError(t, err)
	assert.Equal(t, result.stdout, "This is input!")
	assert.Equal(t, result.prettyCmd, "cat -")
	assert.Empty(t, result.stderr)
}

func TestRunExeFailure(t *testing.T) {
	ctx := context.Background()

	result, err := runExe(ctx, "bash", []string{"-c", "exit 1"}, nil, "")

	assert.Error(t, err)
	assert.Empty(t, result.stdout)
	assert.Equal(t, result.prettyCmd, "bash -c \"exit 1\"")
	assert.Empty(t, result.stderr)
}

func TestRunExeAsyncSuccess(t *testing.T) {
	ctx := context.Background()

	cfg := runExeConf{
		exePath: "cat",
		args:    []string{"-"},
		stdin:   "This is input!",
		env:     nil,
		verbose: true,
	}

	resolve := runExeAsync(ctx, cfg)

	output, err := resolve()

	assert.NoError(t, err)
	msg := "Success running 'cat -', logs follow, if any.\n\nNormal output:\n\nThis is input!"
	assert.Equal(t, output, msg)
}

func TestRunExeAsyncFailure(t *testing.T) {
	ctx := context.Background()

	cfg := runExeConf{
		exePath: "bash",
		args:    []string{"-c", "cat - && echo 'some stderr' >&2 && exit 1"},
		stdin:   "stdin",
		env:     nil,
		verbose: true,
	}

	resolve := runExeAsync(ctx, cfg)

	output, err := resolve()

	assert.Error(t, err)
	msg := "Failure running 'bash -c \"cat - && echo 'some stderr' >&2 && exit 1\"', logs follow."
	msgStderr := "Verbose output:\n\nsome stderr"
	msgStdin := "stdin"
	assert.Contains(t, output, msg)
	assert.Contains(t, output, msgStderr)
	// Stdin is not forwarded.
	assert.NotContains(t, output, msgStdin)
}

func TestNewRunSelfConf(t *testing.T) {
	_, err := newRunSelfConf(
		"some-path", "unknown-command", rootConfigT{}, downloadConfigT{}, serveConfigT{},
	)

	assert.Error(t, err)

	// We only test that we can create a config for each of the expected commands. If we were to
	// test the flags, too, we would be duplicating the implementation mostly.

	type test struct {
		cmd     string
		numArgs int
		stdin   string
	}

	testCases := []test{
		{"list", 8, ""},
		{"serve", 12, ""},
		{"download", 14, ""},
		{"login", 9, "password"},
	}

	for _, tc := range testCases {
		t.Log(tc.cmd)
		cfg, err := newRunSelfConf(
			"some-path",
			tc.cmd,
			rootConfigT{password: "password", verbose: tc.cmd == "login"},
			downloadConfigT{folders: []string{"_ALL_", "-_Gmail_"}},
			serveConfigT{},
		)

		assert.NoError(t, err)
		assert.Equal(t, len(cfg.args), tc.numArgs)
		assert.Equal(t, cfg.stdin, tc.stdin)
		assert.Equal(t, cfg.env, []string{"IGRAB_PASSWORD=password"})
	}
}
