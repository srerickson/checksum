# Checksum  

[![](https://godoc.org/github.com/srerickson/checksum?status.svg)](https://godoc.org/github.com/srerickson/checksum)

Go module for concurrent checksums. Uses `fs.FS` (go v1.16).

## Example

```go
// function called for each complete job
each := func(done checksum.Job) {
    if done.SumString() == "e8c078f0e4ad79b16fcb618a3790c2df" {
        fmt.Println(done.Path())
    }
}
// walk over an fs.FS, doing checksums
err := checksum.Walk(os.DirFS("test/fixture"), each,
    checksum.PipeGos(5),       // 5 go routines
    checksum.PipeAlg(md5.New)) // md5sum

if err != nil {
    log.Fatal(err)
}
// Output: folder1/folder2/sculpture-stone-face-head-888027.jpg
```