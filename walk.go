package checksum

import (
	"errors"
	"io/fs"
)

// JobFunc is a function called for each complete job by Walk(). The funciton is
// called in the same go routine as the call to Walk()
type JobFunc func(Job, error)

// SkipFile is an error returned by a WalkDirFunc to signal that the item in the
// path should not be added to the Pipe
var ErrSkipFile = errors.New(`skip file`)

// DefaultWalkDirFunc is the defult WalkDirFunc used by Walk. It only adds
// regular files to the Pipe.
func DefaultWalkDirFunc(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !d.Type().IsRegular() {
		// don't checksum
		return ErrSkipFile
	}
	return nil
}

func Walk(fsys fs.FS, root string, each JobFunc, opts ...func(*Config)) error {
	p, err := NewPipe(fsys, opts...)
	if err != nil {
		return err
	}
	pip, _ := (p).(*pipe) // for walkDirFunc
	walkErr := make(chan error, 1)
	go func() {
		defer pip.Close()
		defer close(walkErr)
		walk := func(path string, d fs.DirEntry, e error) error {
			if err := pip.conf.walkDirFunc(path, d, e); err != nil {
				if err == ErrSkipFile {
					return nil // continue walk but no checksum
				}
				return err
			}
			return pip.Add(path)
		}
		walkErr <- fs.WalkDir(fsys, root, walk)
	}()
	// complete jobs
	for complete := range pip.Out() {
		each(complete, complete.Err())
	}
	return <-walkErr
}
