package checksum

import (
	"encoding/hex"
	"hash"
	"io"
	"io/fs"
)

// Job is value streamed to/from Walk and Pool
type Job struct {
	path  string   // path to file
	algs  []Alg    // hash constructor function
	valid []byte   // expected checksum (for validation)
	sums  [][]byte // checksum result
	err   error    // any encountered errors
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
	var hashes []hash.Hash
	var writers []io.Writer
	for _, newHash := range j.algs {
		h := newHash()
		hashes = append(hashes, h)
		writers = append(writers, io.Writer(h))
	}
	multi := io.MultiWriter(writers...)
	_, j.err = io.Copy(multi, file)
	if j.err != nil {
		return
	}
	for i := range hashes {
		j.sums = append(j.sums, hashes[i].Sum(nil))
	}
}

// JobAlg is used to set job's checksum algorithm.
// This option may be repeated with different algorithms
// in order to generate multiple chacksums per file.
// Use as a functional argument in Add()
func JobAlg(alg func() hash.Hash) func(*Job) {
	return func(j *Job) {
		j.algs = append(j.algs, alg)
	}
}

// Path returns the path of the job's file
func (j Job) Path() string {
	return j.path
}

// Alg returns the hash functions used in the job
func (j Job) Algs() []Alg {
	return j.algs
}

// Sum returns the first checksum
func (j Job) Sum() []byte {
	if len(j.sums) == 0 {
		return nil
	}
	var s = make([]byte, len(j.sums[0]))
	copy(s, j.sums[0])
	return s
}

// SumString returns the Job's checksum as a hex encoded string
func (j Job) SumString() string {
	return hex.EncodeToString(j.Sum())
}

// Info returns os.FileInfo from the file of a completed Job
func (j Job) Info() fs.FileInfo {
	return j.info
}

func (j Job) Err() error {
	return j.err
}
