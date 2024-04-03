/*
 * Copyright 2023 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package command

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/go-logr/logr"
)

var log logr.Logger

// Run command and grab the default timeout environment variable, if set
func Run(args string, log logr.Logger) (string, error) {
	return run(args, log, nil, nil)
}

// Run command as a specific UID/GID
func RunAs(args string, log logr.Logger, uid, gid uint32) (string, error) {
	return run(args, log, &uid, &gid)
}

// Run with a specific timeout
func RunWithTimeout(args string, timeout int, log logr.Logger) (string, error) {
	return runWithTimeout(args, timeout, log, nil, nil)
}

// Run command as a specific UID/GID with a specific timeout
func RunAsWithTimeout(args string, timeout int, log logr.Logger, uid, gid *uint32) (string, error) {
	return runWithTimeout(args, timeout, log, uid, gid)
}

func run(args string, log logr.Logger, uid, gid *uint32) (string, error) {
	timeoutString, found := os.LookupEnv("NNF_COMMAND_TIMEOUT_SECONDS")
	if found {
		timeout, err := strconv.Atoi(timeoutString)
		if err != nil {
			return "", err
		}

		return runWithTimeout(args, timeout, log, uid, gid)
	}

	return runWithTimeout(args, 0, log, uid, gid)
}

func runWithTimeout(args string, timeout int, log logr.Logger, uid, gid *uint32) (string, error) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()
	}

	var stdout, stderr bytes.Buffer
	shellCmd := exec.CommandContext(ctx, "bash", "-c", args)
	shellCmd.Stdout = &stdout
	shellCmd.Stderr = &stderr

	// If UID/GID are set, then run this command with those
	if uid != nil && gid != nil {
		shellCmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: *uid, Gid: *gid}}
		log.V(1).Info("Command Run", "UID", uid, "GID", gid, "command", args)
	} else {
		log.V(1).Info("Command Run", "command", args)
	}

	err := shellCmd.Run()
	if err != nil {
		return stdout.String(), fmt.Errorf("command: %s - stderr: %s - stdout: %s - error: %w", args, stderr.String(), stdout.String(), err)
	}

	// Command success, return stdout
	return stdout.String(), nil
}
