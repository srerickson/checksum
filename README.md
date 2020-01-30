# Checksum  

[![](https://godoc.org/github.com/srerickson/checksum?status.svg)](https://godoc.org/github.com/srerickson/checksum)

This Go module provides primitives for concurrently generating checksums of files in a cancellable context.

## Example

This example uses Pipe to has files in directory:

```go

pipe := checksum.NewPipe(checksum.WithGoNum(1))
walkErrs := make(chan error, 1)


go func() {
    defer pipe.Close()
    defer close(walkErrs)
    walk := func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.Mode().IsRegular() {
            err := pipe.Add(checksum.Job{Path: path, HashNew: hashNew})
            if err != nil {
                return err
            }
        }
        return nil
    }
    walkErrs <- filepath.Walk(root, walk)
}()
return pipe.Out(), walkErrs


// generate MD5 sum for all regular files in the test/fixture folder
jobs, errs := walk("test/fixture", md5.New)

// print the results
for j := range jobs {
    fmt.Printf("%s: %s\n", j.Path, j.SumString())
}

// check for walk error
if err := <-errs; err != nil {
    fmt.Println(err.Error())
}
// test/fixture/folder1/file.txt: d41d8cd98f00b204e9800998ecf8427e
// test/fixture/folder1/folder2/file2.txt: d41d8cd98f00b204e9800998ecf8427e
// test/fixture/hello.csv: 9d02fa6e9dd9f38327f7b213daa28be6
// test/fixture/folder1/folder2/sculpture-stone-face-head-888027.jpg: e8c078f0e4ad79b16fcb618a3790c2df

```