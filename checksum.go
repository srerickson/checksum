package checksum

// Copyright 2020 Seth R. Erickson
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

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// Pipe is the core object
type Pipe struct {
	out    chan Job // Job results
	in     chan Job // jop input
	wg     sync.WaitGroup
	numGos int // number of goroutines in pool
	ctx    context.Context
}

// Job is value streamed to/from Walk and Pool
type Job struct {
	Path    string           // path to file
	HashNew func() hash.Hash // hash constructor function
	Valid   []byte           // expected checksum (for validation)
	Sum     []byte           // checksum result
	Err     error            // any encountered errors
}

// Sum hashes a file path using hash returned by hashNew
func Sum(path string, hashNew func() hash.Hash) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	hash := hashNew()
	_, err = io.Copy(hash, file)
	if err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
}

// SumString returns job's checksum as a hex encoded string
func (j Job) SumString() string {
	return hex.EncodeToString(j.Sum)
}

// IsValid returns whether the job's checksum matches expected value
func (j Job) IsValid() bool {
	return j.Sum != nil && bytes.Equal(j.Sum, j.Valid)
}

// WithContext is used to add a Context to Walk and Pool
func WithContext(ctx context.Context) func(*Pipe) {
	return func(p *Pipe) {
		p.ctx = ctx
	}
}

// WithGoNum is used set the number of goroutines in the Pool
func WithGoNum(n int) func(*Pipe) {
	return func(p *Pipe) {
		if n < 1 {
			n = 1
		}
		p.numGos = n
	}
}

// NewPipe returns a new Pipe
func NewPipe(opts ...func(*Pipe)) *Pipe {
	pipe := &Pipe{
		in:     make(chan Job),
		out:    make(chan Job),
		ctx:    context.Background(),
		numGos: runtime.GOMAXPROCS(0),
	}
	for _, option := range opts {
		option(pipe)
	}
	pipe.wg.Add(pipe.numGos)
	for i := 0; i < pipe.numGos; i++ {
		go func() {
			defer pipe.wg.Done()
			for job := range pipe.in {
				select {
				case <-pipe.ctx.Done():
					continue // drain input channel w/o doing Sum
				default:
					if job.Err == nil {
						job.Sum, job.Err = Sum(job.Path, job.HashNew)
					}
					pipe.out <- job
				}
			}
		}()
	}
	go func() {
		pipe.wg.Wait()
		close(pipe.out)
	}()
	return pipe
}

// Out returns the Pipe's recieve-only channel of Job results
func (p *Pipe) Out() <-chan Job {
	return p.out
}

// Close closes the Pipes input channel
func (p *Pipe) Close() {
	close(p.in)
}

// Add adds a Job to the Pipe input. Returns error if the Pipe context is canceled.
func (p *Pipe) Add(j Job) error {
	select {
	case <-p.ctx.Done():
		return errors.New(`walk canceled`)
	default:
		p.in <- j
	}
	return nil
}

//Walk concurrently calculates checksums of regular files in a director
func (p *Pipe) Walk(dir string, hashNew func() hash.Hash) <-chan error {
	errs := make(chan error, 1)
	go func() {
		defer p.Close()
		walk := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				err := p.Add(Job{Path: path, HashNew: hashNew})
				if err != nil {
					return err
				}
			}
			return nil
		}
		errs <- filepath.Walk(dir, walk)
	}()
	return errs
}
