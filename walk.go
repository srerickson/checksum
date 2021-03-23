package checksum

import "io/fs"

type JobFunc func(Job)

func Walk(fsys fs.FS, root string, each JobFunc, opts ...func(*Pipe)) error {
	pipe := NewPipe(fsys, opts...)
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
		each(complete)
	}
	return <-walkErr
}
