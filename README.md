# Checksum  

[![](https://godoc.org/github.com/srerickson/checksum?status.svg)](https://godoc.org/github.com/srerickson/checksum)

Go module for concurrent checksums. Uses `fs.FS` (go v1.16).

## Example

```go
// called for each complete job
each := func(done checksum.Job, err error) {
    if err != nil {
        // handle error
        log.Println(err)
        return
    }
    if done.SumString(checksum.MD5) == "e8c078f0e4ad79b16fcb618a3790c2df" {
        fmt.Println(done.SumString(checksum.SHA1))
    }
}
err := checksum.Walk(os.DirFS("test/fixture"), ".", each,
    checksum.WithGos(5), // 5 go routines
    checksum.WithMD5(),  // md5sum
    checksum.WithSHA1()) // sha1

if err != nil {
    fmt.Println(err)
}
// Output: a0556088c3b6a78b2d8ef7b318cfca54589f68c0
```