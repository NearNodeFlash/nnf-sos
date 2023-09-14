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
	"time"

	"github.com/go-logr/logr"
)

var log logr.Logger

func RunWithTimeout(args string, timeout int) (string, error) {

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

	log.Info("Run", "command", args)

	err := shellCmd.Run()
	if err != nil {
		return stdout.String(), fmt.Errorf("command: %s - stderr: %s - stdout: %s - error: %w", args, stderr.String(), stdout.String(), err)
	}

	// Command success, return stdout
	return stdout.String(), nil
}

func Run(args string) (string, error) {
	timeoutString, found := os.LookupEnv("NNF_COMMAND_TIMEOUT_SECONDS")
	if found {
		timeout, err := strconv.Atoi(timeoutString)
		if err != nil {
			return "", err
		}

		return RunWithTimeout(args, timeout)
	}

	return RunWithTimeout(args, 0)
}
