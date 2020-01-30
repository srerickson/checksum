package checksum_test

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"testing"

	"github.com/srerickson/checksum"
)

var testMD5Sums = map[string]string{
	"test/fixture/folder1/folder2/sculpture-stone-face-head-888027.jpg": "e8c078f0e4ad79b16fcb618a3790c2df",
	"test/fixture/folder1/folder2/file2.txt":                            "d41d8cd98f00b204e9800998ecf8427e",
	"test/fixture/folder1/file.txt":                                     "d41d8cd98f00b204e9800998ecf8427e",
	"test/fixture/hello.csv":                                            "9d02fa6e9dd9f38327f7b213daa28be6",
}

// func TestWalk(t *testing.T) {
// 	pipe := checksum.NewPipe()
// 	errs := pipe.Walk(`test/fixture`, md5.New)
// 	for j := range pipe.Out() {
// 		if j.Err != nil {
// 			t.Error(j.Err)
// 		}
// 	}
// 	err := <-errs
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}
// }

// func TestWalk1Chan(t *testing.T) {
// 	pipe := checksum.NewPipe(checksum.WithGoNum(1))
// 	errs := pipe.Walk(`test/fixture`, md5.New)
// 	for j := range pipe.Out() {
// 		if j.Err != nil {
// 			t.Error(j.Err)
// 		}
// 	}
// 	err := <-errs
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}
// }

func TestContextCancel(t *testing.T) {
	errs := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	pipe := checksum.NewPipe(checksum.WithContext(ctx))
	go func() {
		defer pipe.Close()
		j1 := checksum.Job{Path: "nofile1", HashNew: md5.New}
		j2 := checksum.Job{Path: "nofile2", HashNew: md5.New}
		pipe.Add(j1)
		cancel() // <-- cancel the context
		errs <- pipe.Add(j2)
	}()
	numResults := 0
	for range pipe.Out() {
		numResults++
	}
	if numResults > 1 {
		t.Error(`expected only one result`)
	}
	if <-errs == nil {
		t.Error(`expected canceled context error`)
	}
}

func TestPipeErr(t *testing.T) {
	pipe := checksum.NewPipe(checksum.WithGoNum(2))
	go func() {
		defer pipe.Close()
		pipe.Add(checksum.Job{
			Path:    `nofile`,
			HashNew: md5.New,
		})
	}()
	result := <-pipe.Out()
	if result.Err == nil {
		t.Error(`expected read error`)
	}
	_, alive := <-pipe.Out()
	if alive {
		t.Error(`expected closed chan`)
	}
}

func TestValidate(t *testing.T) {
	pipe := checksum.NewPipe()
	go func() {
		defer pipe.Close()
		for path, sum := range testMD5Sums {
			sumBytes, _ := hex.DecodeString(sum)
			pipe.Add(checksum.Job{
				Path:    path,
				Valid:   sumBytes,
				HashNew: md5.New})
		}
	}()
	numResults := 0
	for j := range pipe.Out() {
		numResults++
		if !j.IsValid() {
			t.Error(`expected valid result`)
		}
	}
	if numResults != len(testMD5Sums) {
		t.Errorf("expected %d results, got %d", len(testMD5Sums), numResults)
	}

}

func ExampleWalk() {
	// walk creates a Pipe, walks the filetree at root, and adds checksum jobs to the pipe
	// for each regular file.
	walk := func(root string, hashNew func() hash.Hash) (<-chan checksum.Job, <-chan error) {
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
	}
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
}
