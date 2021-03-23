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
	"errors"
	"hash"
	"io/fs"
	"sync"
)

// Alg is a function that returns a hash.Hash
type Alg func() hash.Hash

// Pipe is a checksum worker pool. It has an input channel and an output channel.
// Add jobs to the input channel with Add() and receive results with Out(). Typically
// these are called in different go routines.
type Pipe struct {
	conf Config   // common config options
	fsys fs.FS    // the pipe's jobs are scoped to the fs
	in   chan Job // jop input
	out  chan Job // job results
}

// NewPipe returns a new Pipe scoped to dir
// in the given FS. Without options, the Pipe has defaults:
// - PipeGos(runtime.GOMAXPROCS(0))
// - PipeCtx(context.Background())
// - no checksums are defined
func NewPipe(fsys fs.FS, opts ...func(*Config)) (*Pipe, error) {
	pipe := &Pipe{
		fsys: fsys,
		in:   make(chan Job),
		out:  make(chan Job),
		conf: defaultConfig(),
	}
	for _, option := range opts {
		option(&pipe.conf)
	}
	if len(pipe.conf.algs) == 0 {
		return nil, errors.New(`checksum algorithms not defined`)
	}

	var wg sync.WaitGroup
	for i := 0; i < pipe.conf.numGos; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range pipe.in {
				select {
				case <-pipe.conf.ctx.Done():
					continue // clear input channel
				default:
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
	return pipe, nil
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
func (p *Pipe) Add(path string) error {
	j := Job{
		path: path,
		fs:   p.fsys,
		algs: p.conf.algs,
	}
	select {
	case <-p.conf.ctx.Done():
		return errors.New(`walk canceled`)
	default:
		p.in <- j
	}
	return nil
}
