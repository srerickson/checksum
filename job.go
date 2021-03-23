package checksum

import (
	"bytes"
	"encoding/hex"
	"hash"
	"io"
	"io/fs"
)

// Job is value streamed to/from Walk and Pool
type Job struct {
	path  string           // path to file
	alg   func() hash.Hash // hash constructor function
	valid []byte           // expected checksum (for validation)
	sum   []byte           // checksum result
	err   error            // any encountered errors
	fs    fs.FS
	info  fs.FileInfo
}

// NewJob returns a new checksum job for the path
func newJob(path string, opts ...func(*Job)) Job {
	j := Job{path: path}
	for _, opt := range opts {
		opt(&j)
	}
	return j
}

// do does the job
func (j *Job) do() {
	if j.err != nil {
		return
	}
	var file fs.File
	file, j.err = j.fs.Open(j.path)
	if j.err != nil {
		return
	}
	defer file.Close()
	j.info, j.err = file.Stat()
	if j.err != nil {
		return
	}
	hash := j.alg()
	_, j.err = io.Copy(hash, file)
	if j.err != nil {
		return
	}
	j.sum = hash.Sum(nil)
}

// JobAlg is used to set job's checksum algorithm.
// Use as a functional argument in Add()
func JobAlg(alg func() hash.Hash) func(*Job) {
	return func(j *Job) {
		j.alg = alg
	}
}

// JobSum is used to set a job's expected checksum.
// Use as a functional argument in Add()
func JobSum(sum []byte) func(*Job) {
	return func(j *Job) {
		j.valid = sum
	}
}

// Path returns the path of the job's file
func (j Job) Path() string {
	return j.path
}

// Alg returns the hash function used in the job
func (j Job) Alg() func() hash.Hash {
	return j.alg
}

// Sum returns the checksum
func (j Job) Sum() []byte {
	var s []byte
	copy(s, j.sum)
	return s
}

// Expected returns the expected checksum (for validation)
func (j Job) Expected() []byte {
	var s []byte
	copy(s, j.sum)
	return s
}

// SumString returns the Job's checksum as a hex encoded string
func (j Job) SumString() string {
	return hex.EncodeToString(j.sum)
}

// IsValid returns whether the Job's checksum matches expected value
func (j Job) IsValid() bool {
	return j.sum != nil && bytes.Equal(j.sum, j.valid)
}

// Info returns os.FileInfo from the file of a completed Job
func (j Job) Info() fs.FileInfo {
	return j.info
}

func (j Job) Err() error {
	return j.err
}
