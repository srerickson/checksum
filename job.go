package checksum

import (
	"bytes"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

// Job is value streamed to/from Walk and Pool
type Job struct {
	Path    string           // path to file
	HashNew func() hash.Hash // hash constructor function
	Valid   []byte           // expected checksum (for validation)
	Sum     []byte           // checksum result
	info    os.FileInfo
	Err     error // any encountered errors
}

// SumString returns the Job's checksum as a hex encoded string
func (j Job) SumString() string {
	return hex.EncodeToString(j.Sum)
}

// IsValid returns whether the Job's checksum matches expected value
func (j Job) IsValid() bool {
	return j.Sum != nil && bytes.Equal(j.Sum, j.Valid)
}

// Info returns os.FileInfo from the file of a completed Job
func (j Job) Info() os.FileInfo {
	return j.info
}

// Do does the job
func (j *Job) Do() {
	if j.Err != nil {
		return
	}
	var file *os.File
	file, j.Err = os.Open(j.Path)
	if j.Err != nil {
		return
	}
	defer file.Close()
	j.info, j.Err = file.Stat()
	if j.Err != nil {
		return
	}
	hash := j.HashNew()
	_, j.Err = io.Copy(hash, file)
	if j.Err != nil {
		return
	}
	j.Sum = hash.Sum(nil)
}
