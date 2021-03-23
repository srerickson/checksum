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
	"crypto/sha256"
	"errors"
	"hash"
	"io/fs"
	"runtime"
	"sync"
)

type JobFunc func(Job)

func Walk(dir fs.FS, each JobFunc, opts ...func(*Pipe)) error {
	pipe := NewPipe(dir, opts...)
	walkErr := make(chan error, 1)
	go func() {
		defer pipe.Close()
		defer close(walkErr)
		walk := func(path string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if info.Type().IsRegular() {
				err := pipe.Add(path)
				if err != nil {
					return err
				}
			}
			return nil
		}
		walkErr <- fs.WalkDir(dir, `.`, walk)
	}()
	// complete jobs
	for complete := range pipe.Out() {
		each(complete)
	}
	return <-walkErr
}

// Pipe is a checksum worker pool. It has an input channel and an output channel.
// Add jobs to the input channel with Add() and receive results with Out(). Typically
// these operations happen on different go routines.
type Pipe struct {
	in     chan Job // jop input
	out    chan Job // job results
	numGos int      // number of goroutines in pool
	ctx    context.Context
	alg    func() hash.Hash
}

// PipeGos is used set the number of goroutines used by a Pipe.
// Used as an optional argument for NewPipe().
func PipeGos(n int) func(*Pipe) {
	return func(p *Pipe) {
		if n < 1 {
			n = 1
		}
		p.numGos = n
	}
}

// PipeCtx is used set a Pipe's context.
// Used as an optional argument for NewPipe().
func PipeCtx(ctx context.Context) func(*Pipe) {
	return func(p *Pipe) {
		p.ctx = ctx
	}
}

// PipeAlg sets the default hash alforithm used in the Pipe.
// Used as an optional argument for NewPipe().
func PipeAlg(alg func() hash.Hash) func(*Pipe) {
	return func(p *Pipe) {
		p.alg = alg
	}
}

// NewPipe returns a new Pipe scoped to dir
// in the given FS. Without options, the Pipe has defaults:
// - PipeGos(runtime.GOMAXPROCS(0))
// - PipeCtx(context.Background())
// - PipeAlg(sha256.New)
func NewPipe(dir fs.FS, opts ...func(*Pipe)) *Pipe {
	pipe := &Pipe{
		in:     make(chan Job),
		out:    make(chan Job),
		ctx:    context.Background(),
		numGos: runtime.GOMAXPROCS(0),
		alg:    sha256.New, // default alg
	}
	for _, option := range opts {
		option(pipe)
	}
	var wg sync.WaitGroup
	for i := 0; i < pipe.numGos; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range pipe.in {
				select {
				case <-pipe.ctx.Done():
					continue // clear input channel
				default:
					// configure and run the job
					job.fs = dir
					if job.alg == nil {
						job.alg = pipe.alg
					}
					job.do()
					pipe.out <- job
				}
			}
		}()
	}
	go func() {
		wg.Wait()
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

// Add adds a checksum job for path to the Pipe. It returns an
// error if the Pipe context is canceled. Typically, to avoid
// deadlocks, Add is  called in a separate goroutine than was
// used to create the pipe with NewPipe().
func (p *Pipe) Add(path string, opts ...func(*Job)) error {
	j := newJob(path, opts...)
	select {
	case <-p.ctx.Done():
		return errors.New(`walk canceled`)
	default:
		p.in <- j
	}
	return nil
}
