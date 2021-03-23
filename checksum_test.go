package checksum_test

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/srerickson/checksum"
)

var testMD5Sums = map[string]string{
	"test/fixture/folder1/folder2/sculpture-stone-face-head-888027.jpg": "e8c078f0e4ad79b16fcb618a3790c2df",
	"test/fixture/folder1/folder2/file2.txt":                            "d41d8cd98f00b204e9800998ecf8427e",
	"test/fixture/folder1/file.txt":                                     "d41d8cd98f00b204e9800998ecf8427e",
	"test/fixture/hello.csv":                                            "9d02fa6e9dd9f38327f7b213daa28be6",
}

func TestContextCancel(t *testing.T) {
	errs := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	dir := os.DirFS(`.`)
	pipe := checksum.NewPipe(dir, checksum.PipeCtx(ctx))
	go func() {
		defer pipe.Close()
		pipe.Add(`nofile1`, checksum.JobAlg(md5.New))
		cancel() // <-- cancel the context
		errs <- pipe.Add(`nofile2`, checksum.JobAlg(md5.New))
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
	dir := os.DirFS(`.`)
	pipe := checksum.NewPipe(dir, checksum.PipeGos(2), checksum.PipeAlg(md5.New))
	go func() {
		defer pipe.Close()
		pipe.Add(`nofile`)
	}()
	result := <-pipe.Out()
	if result.Err() == nil {
		t.Error(`expected read error`)
	}
	_, alive := <-pipe.Out()
	if alive {
		t.Error(`expected closed chan`)
	}
}

func TestValidate(t *testing.T) {
	dir := os.DirFS(`.`)
	pipe := checksum.NewPipe(dir, checksum.PipeGos(3), checksum.PipeAlg(md5.New))
	go func() {
		defer pipe.Close()
		for path, sum := range testMD5Sums {
			sumBytes, _ := hex.DecodeString(sum)
			pipe.Add(path, checksum.JobSum(sumBytes))
		}
	}()
	numResults := 0
	for j := range pipe.Out() {
		numResults++
		if !j.IsValid() {
			t.Error(`expected valid result`)
		}
		if j.Info().Mode() != 0644 {
			t.Error(`expected 0644 FileMode`)
		}
	}
	if numResults != len(testMD5Sums) {
		t.Errorf("expected %d results, got %d", len(testMD5Sums), numResults)
	}

}

func ExampleWalk() {
	// called for each complete job
	each := func(done checksum.Job) {
		if done.SumString() == "e8c078f0e4ad79b16fcb618a3790c2df" {
			fmt.Println(done.Path())
		}
	}
	err := checksum.Walk(os.DirFS("test/fixture"), each,
		checksum.PipeGos(5),       // 5 go routines
		checksum.PipeAlg(md5.New)) // md5sumccccccucjiulllcdgccelnnevggubdfrtflddgickcur

	if err != nil {
		log.Fatal(err)
	}
	// Output: folder1/folder2/sculpture-stone-face-head-888027.jpg
}
