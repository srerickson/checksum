package checksum_test

import (
	"context"
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
	pipe, _ := checksum.NewPipe(dir, checksum.WithCtx(ctx), checksum.WithMD5())
	go func() {
		defer pipe.Close()
		pipe.Add(`nofile1`)
		cancel() // <-- cancel the context
		errs <- pipe.Add(`nofile2`)
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
	pipe, err := checksum.NewPipe(dir, checksum.WithGos(2), checksum.WithMD5())
	if err != nil {
		t.Fatal(err)
	}
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
	pipe, err := checksum.NewPipe(dir, checksum.WithGos(3), checksum.WithMD5())
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		defer pipe.Close()
		for path := range testMD5Sums {
			pipe.Add(path)
		}
	}()
	numResults := 0
	for j := range pipe.Out() {
		numResults++
		got := j.SumString(checksum.MD5)
		expected := testMD5Sums[j.Path()]
		if got != expected {
			t.Errorf(`expected %s, got %s for %s`, expected, got, j.Path())
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
}
