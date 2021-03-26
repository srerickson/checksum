package checksum

import (
	"errors"
	"fmt"
	"io/fs"
)

// JobFunc is a function called for each complete job by Walk(). The funciton is
// called in the same go routine as the call to Walk(). If JobFunc() returns an
// error, Walk will close the pipe and JobFunc will not be called again
type JobFunc func(Job, error) error

// WalkErr combines the two kinds of errors that Walk() may need to report
// in one object
type WalkErr struct {
	WalkDirErr error // error returned from WalkDir
	JobFuncErr error // error returned from JobFunc
}

// Error implements error interface for WalkErr
func (we *WalkErr) Error() string {
	return fmt.Sprintf(`WalkDirErr: %s; JobErr: %s`, we.WalkDirErr.Error(), we.JobFuncErr.Error())
}

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
	walkErrChan := make(chan error, 1)
	go func() {
		defer pip.Close()
		defer close(walkErrChan)
		walk := func(path string, d fs.DirEntry, e error) error {
			if err := pip.conf.walkDirFunc(path, d, e); err != nil {
				if err == ErrSkipFile {
					return nil // continue walk but no checksum
				}
				return err
			}
			return pip.Add(path)
		}
		walkErrChan <- fs.WalkDir(fsys, root, walk)
	}()

	// process job callbacks and capture errors
	var jobFuncErr error
	for complete := range pip.Out() {
		if jobFuncErr == nil {
			jobFuncErr = each(complete, complete.Err())
		}
	}
	walkErr := <-walkErrChan
	if jobFuncErr != nil || walkErr != nil {
		return &WalkErr{
			WalkDirErr: walkErr,
			JobFuncErr: jobFuncErr,
		}
	}
	return nil
}
