// Package checksum provides primitives for concurrently checksuming files
// in a cancellable context.
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
	"context"
	"errors"
	"hash"
	"io"
	"os"
	"runtime"
	"sync"
)

// Pipe is the central type provided by the package. Pipes are created with
// NewPipe() and Jobs can be added to Pipes with Add(). Results are sent through
// the channel returned by Out().
type Pipe struct {
	out    chan Job // Job results
	in     chan Job // jop input
	wg     sync.WaitGroup
	numGos int // number of goroutines in pool
	ctx    context.Context
}

// Sum returns the checksum of the file at path using the hash returned by
// hashNew. An error is returned if the file could not be opened or read.
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

// WithGoNum is used set the number of goroutines used by a Pipe to
// generate checksums. It is meant to be used as an argument in NewPipe().
func WithGoNum(n int) func(*Pipe) {
	return func(p *Pipe) {
		if n < 1 {
			n = 1
		}
		p.numGos = n
	}
}

// WithContext is used set a Pipe's context. It is meant to be used
// as an argument ing NewPipe().
func WithContext(ctx context.Context) func(*Pipe) {
	return func(p *Pipe) {
		p.ctx = ctx
	}
}

// NewPipe returns a new Pipe. Without options, the Pipe is created
// with runtime.GOMAXPROCS(0) goroutines and the context.Background()
// context.
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
					job.Do()
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

// Close closes the Pipe's input channel.
func (p *Pipe) Close() {
	close(p.in)
}

// Add adds a Job to the Pipe. It returns an error if the Pipe context is
// canceled. Typically, to avoid deadlocks, Add should be called in a separate
// goroutine than was used to create the pipe with NewPipe().
func (p *Pipe) Add(j Job) error {
	select {
	case <-p.ctx.Done():
		return errors.New(`walk canceled`)
	default:
		p.in <- j
	}
	return nil
}
