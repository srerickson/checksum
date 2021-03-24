package checksum

import (
	"encoding/hex"
	"hash"
	"io"
	"io/fs"
)

// Job is value streamed to/from Walk and Pool
type Job struct {
	path string                      // path to file
	algs map[string]func() hash.Hash // hash constructor function
	sums map[string][]byte           // checksum result
	err  error                       // any encountered errors
	fs   fs.FS
	info fs.FileInfo
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
	var hashes = make(map[string]hash.Hash)
	var writers []io.Writer
	for name, newHash := range j.algs {
		h := newHash()
		hashes[name] = h
		writers = append(writers, io.Writer(h))
	}
	multi := io.MultiWriter(writers...)
	_, j.err = io.Copy(multi, file)
	if j.err != nil {
		return
	}
	j.sums = make(map[string][]byte)
	for name, h := range hashes {
		j.sums[name] = h.Sum(nil)
	}
}

// Path returns the path of the job's file
func (j Job) Path() string {
	return j.path
}

// Sum returns the first checksum
func (j Job) Sum(name string) []byte {
	if j.sums == nil || j.sums[name] == nil {
		return nil
	}
	var s = make([]byte, len(j.sums[name]))
	copy(s, j.sums[name])
	return s
}

// SumString returns the Job's checksum as a hex encoded string
func (j Job) SumString(name string) string {
	return hex.EncodeToString(j.Sum(name))
}

// Info returns os.FileInfo from the file of a completed Job
func (j Job) Info() fs.FileInfo {
	return j.info
}

func (j Job) Err() error {
	return j.err
}
