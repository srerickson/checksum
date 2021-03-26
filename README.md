# Checksum  

[![](https://godoc.org/github.com/srerickson/checksum?status.svg)](https://godoc.org/github.com/srerickson/checksum)

Go module for concurrent checksums. Uses `fs.FS` (go v1.16).

## Examples

### Duplicates

An example program that identifies all identical files under a directory

```go
// examples/duplicates/duplicates.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/srerickson/checksum"
)

func main() {
	flag.Parse()
	dir := flag.Arg(0)
	if dir == "" {
		log.Fatal(`required argument: the directory to checksum`)
	}
	dirFS := os.DirFS(dir)
	duplicates := make(map[string][]string)
	each := func(j checksum.Job, err error) error {
		// Callback function called for each complete job.
		// Returning prevents future calls to each().
		if err != nil {
			return err
		}
		sum, err := j.SumString(checksum.MD5)
		if err != nil {
			return err
		}
		duplicates[sum] = append(duplicates[sum], j.Path())
		return nil
	}
	err := checksum.Walk(dirFS, `.`, each, checksum.WithMD5())
	if err != nil {
		log.Fatal(err)
	}
	count := 0
	for sum, paths := range duplicates {
		if len(paths) > 1 {
			fmt.Printf("[%s]: %s\n", sum, strings.Join(paths, ", "))
			count++
		}
	}
	if count == 0 {
		fmt.Println(`no dupliactes found`)
	}
}
```