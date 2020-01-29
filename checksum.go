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

// Config holds config options for both Walk and Pool
type Config struct {
	poolSize int // number of goroutines in pool
	ctx      context.Context
}

// Job is value streamed to/from Walk and Pool
type Job struct {
	Path    string           // path to file
	HashNew func() hash.Hash // hash constructor function
	Valid   []byte           // expected checksum (for validation)
	sum     []byte           // checksum result
	err     error            // any encountered errors
}

// Sum hashes a file a path using hash returned by hashNew
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

// Sum returns job's checksum
func (j Job) Sum() []byte {
	return j.sum
}

// SumString returns job's checksum as a hex encoded string
func (j Job) SumString() string {
	return hex.EncodeToString(j.sum)
}

// Err returns the error associated with the job (if any)
func (j Job) Err() error {
	return j.err
}

// IsValid returns whether the job's checksum matches expected value
func (j Job) IsValid() bool {
	return j.sum != nil && bytes.Equal(j.sum, j.Valid)
}

// default configs for Pool and Walk
func defaultConfig() *Config {
	return &Config{
		poolSize: runtime.GOMAXPROCS(0),
		ctx:      context.Background(),
	}
}

// WithContext is used to add a Context to Walk and Pool
func WithContext(ctx context.Context) func(*Config) {
	return func(c *Config) {
		c.ctx = ctx
	}
}

// WithPoolSize is used set the number of goroutines in the Pool
func WithPoolSize(size int) func(*Config) {
	return func(c *Config) {
		if size < 1 {
			size = 1
		}
		c.poolSize = size
	}
}

// Pool processes checksum jobs concurrently
func Pool(jobsIn <-chan Job, opts ...func(*Config)) <-chan Job {
	var wg sync.WaitGroup
	jobsOut := make(chan Job)
	config := defaultConfig()
	for _, option := range opts {
		option(config)
	}
	wg.Add(config.poolSize)
	for i := 0; i < config.poolSize; i++ {
		go func() {
			defer wg.Done()
			for job := range jobsIn {
				select {
				case <-config.ctx.Done():
					return
				default:
					if job.err == nil {
						job.sum, job.err = Sum(job.Path, job.HashNew)
					}
					jobsOut <- job
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(jobsOut)
	}()
	return jobsOut
}

// Walk concurrently calculates checksums of regular files in a director
func Walk(dir string, hashNew func() hash.Hash, opts ...func(*Config)) (<-chan Job, <-chan error) {
	config := defaultConfig()
	for _, option := range opts {
		option(config)
	}
	jobsIn := make(chan Job)
	done := make(chan error, 1)
	// walk files in dir
	go func() {
		defer close(jobsIn)
		defer close(done)
		walk := func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				select {
				case <-config.ctx.Done():
					return errors.New(`walk canceled`)
				default:
					jobsIn <- Job{Path: p, HashNew: hashNew}
				}
			}
			return nil
		}
		done <- filepath.Walk(dir, walk)
	}()
	return Pool(jobsIn, WithContext(config.ctx), WithPoolSize(config.poolSize)), done
}
