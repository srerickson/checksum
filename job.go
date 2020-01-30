package checksum

import (
	"bytes"
	"encoding/hex"
	"hash"
)

// Job is value streamed to/from Walk and Pool
type Job struct {
	Path    string           // path to file
	HashNew func() hash.Hash // hash constructor function
	Valid   []byte           // expected checksum (for validation)
	Sum     []byte           // checksum result
	Err     error            // any encountered errors
}

// SumString returns job's checksum as a hex encoded string
func (j Job) SumString() string {
	return hex.EncodeToString(j.Sum)
}

// IsValid returns whether the job's checksum matches expected value
func (j Job) IsValid() bool {
	return j.Sum != nil && bytes.Equal(j.Sum, j.Valid)
}
