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
	"os"
	"syscall"

	"github.com/mholt/archiver/v3"
	"github.com/tetratelabs/log"

	"github.com/tetratelabs/getenvoy/pkg/binary"
)

func (r *Runtime) handleTermination() {
	cmd := r.cmd

	defer interrupt(cmd.Process) // Ensure the SIGINT forwards to Envoy even if a pre-termination hook panics

	log.Infof("GetEnvoy process (PID=%d) received SIGINT", os.Getpid())
	// Execute all registered preTermination functions
	for _, f := range r.preTermination {
		if err := f(r); err != nil {
			log.Error(err.Error())
		}
	}

	interrupt(cmd.Process)
}

func interrupt(p *os.Process) {
	log.Infof("Sending Envoy process (PID=%d) SIGINT", p.Pid)
	_ = p.Signal(syscall.SIGINT)
}

func (r *Runtime) handlePostTermination() error {
	for _, f := range r.postTermination {
		if err := f(r); err != nil {
			log.Errorf("failed to handle post termination: %v", err)
		}
	}

	// Tar up the debug data and clean up
	if err := archiver.Archive([]string{r.DebugStore()}, r.DebugStore()+".tar.gz"); err != nil {
		return fmt.Errorf("unable to archive debug store directory %v: %v", r.DebugStore(), err)
	}
	return os.RemoveAll(r.DebugStore())
}

// RegisterPreTermination registers the passed functions to be run after Envoy has started
// and just before GetEnvoy instructs Envoy to terminate
func (r *Runtime) RegisterPreTermination(f ...func(binary.Runner) error) {
	r.preTermination = append(r.preTermination, f...)
}

// RegisterPostTermination registers the passed functions to be run after Envoy has terminated
// and just before GetEnvoy archives the debug directory.
func (r *Runtime) RegisterPostTermination(f ...func(binary.Runner) error) {
	r.postTermination = append(r.postTermination, f...)
}
