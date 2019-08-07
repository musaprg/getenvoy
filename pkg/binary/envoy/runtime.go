// Copyright 2019 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package envoy

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/tetratelabs/getenvoy/pkg/binary"
)

// NewRuntime creates a new Runtime with the local file storage set to the home directory
func NewRuntime(options ...func(*Runtime)) (binary.FetchRunner, error) {
	usrDir, err := homedir.Dir()
	local := filepath.Join(usrDir, ".getenvoy")
	runtime := &Runtime{
		fetcher:        fetcher{local},
		wg:             &sync.WaitGroup{},
		AdminEndpoint:  "localhost:15001",
		signals:        make(chan os.Signal),
		preStart:       make([]func(binary.Runner) error, 0),
		preTermination: make([]func(binary.Runner) error, 0),
	}
	for _, option := range options {
		option(runtime)
	}
	return runtime, err
}

type fetcher struct {
	store string
}

// Runtime manages an Envoy lifecycle including fetching (if necessary) and running
type Runtime struct {
	fetcher

	debugDir      string
	AdminEndpoint string

	cmd *exec.Cmd
	wg  *sync.WaitGroup

	signals chan os.Signal

	preStart       []func(binary.Runner) error
	preTermination []func(binary.Runner) error
}

// Status indicates the state of the child process
func (r *Runtime) Status() int {
	switch {
	case r.cmd == nil, r.cmd.Process == nil:
		return binary.StatusStarting
	case r.cmd.ProcessState == nil:
		if r.envoyReady() {
			return binary.StatusReady
		}
		return binary.StatusStarted
	default:
		return binary.StatusTerminated
	}
}

func (r *Runtime) envoyReady() bool {
	resp, err := http.Get(fmt.Sprintf("http://%v/ready", r.AdminEndpoint))
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK
}

// Wait blocks until the child process reaches the state passed
// Note: It does not guarantee that it is in the specified state just that it has reached it
func (r *Runtime) Wait(state int) {
	for r.Status() < state {
		// This is a call to a function to allow the goroutine to be preempted for garbage collection
		func() { time.Sleep(time.Millisecond) }()
	}
}

// SendSignal sends a signal to the parent process
func (r *Runtime) SendSignal(s os.Signal) {
	r.signals <- s
}
