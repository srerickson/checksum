package checksum

import "io/fs"

type JobFunc func(Job, error)

func Walk(fsys fs.FS, root string, each JobFunc, opts ...func(*Config)) error {
	pipe, err := NewPipe(fsys, opts...)
	if err != nil {
		return err
	}
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
		walkErr <- fs.WalkDir(fsys, root, walk)
	}()
	// complete jobs
	for complete := range pipe.Out() {
		each(complete, complete.Err())
	}
	return <-walkErr
}
