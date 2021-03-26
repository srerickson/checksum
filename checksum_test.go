package checksum_test

import (
	"context"
	"errors"
	"fmt"
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
		cancel() // <-- Add() should return err after this
		errs <- pipe.Add(`nofile2`)
	}()
	var numResults int
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
	pipe, err := checksum.NewPipe(dir, checksum.WithGos(1), checksum.WithMD5())
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		defer pipe.Close()
		pipe.Add(`nofile`) // doesn't exist
		pipe.Add(`.`)      // not a regular file
	}()
	for range []int{1, 2} {
		result := <-pipe.Out()
		if result.Err() == nil {
			t.Error(`expected error`)
		}
	}
	_, alive := <-pipe.Out()
	if alive {
		t.Error(`expected closed chan`)
	}
	//pipe.Add(`file`) // panic
}

func TestValidate(t *testing.T) {
	dir := os.DirFS(`.`)

	// map of file sha512 from walk
	shas := make(map[string]string)
	each := func(j checksum.Job, err error) error {
		if err != nil {
			return err
		}
		shas[j.Path()] = j.SumString(checksum.SHA512)
		return nil
	}
	err := checksum.Walk(dir, `.`, each, checksum.WithSHA512())
	if err != nil {
		t.Fatal(err)
	}

	// a pipe with multiple checksums
	pipe, err := checksum.NewPipe(dir,
		checksum.WithGos(3),
		checksum.WithMD5(),
		checksum.WithSHA512())
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		defer pipe.Close()
		for path := range testMD5Sums {
			pipe.Add(path)
		}
	}()
	var numResults int
	for j := range pipe.Out() {
		numResults++
		if err := j.Err(); err != nil {
			t.Error(err)
			continue
		}
		gotMD5 := j.SumString(checksum.MD5)
		gotSHA := j.SumString(checksum.SHA512)
		wantSHA := shas[j.Path()]
		wantMD5 := testMD5Sums[j.Path()]
		if gotMD5 != wantMD5 {
			t.Errorf(`expected MD5 %s, got %s for %s`, wantMD5, gotMD5, j.Path())
		}
		if gotSHA != wantSHA {
			t.Errorf(`expected SHA512 %s, got %s for %s`, wantSHA, gotSHA, j.Path())
		}
	}
	if numResults != len(testMD5Sums) {
		t.Errorf("expected %d results, got %d", len(testMD5Sums), numResults)
	}
}

func TestWalkErr(t *testing.T) {
	expectedErr := errors.New(`stop`)

	// called for each complete job
	each := func(done checksum.Job, err error) error {
		if done.SumString(checksum.MD5) == "e8c078f0e4ad79b16fcb618a3790c2df" {
			return expectedErr
		}
		return nil
	}
	err := checksum.Walk(os.DirFS("test/fixture"), ".",
		each, checksum.WithMD5())

	walkErr, ok := err.(*checksum.WalkErr)
	if !ok {
		t.Error(`expected checksum.WalkErr`)
	}
	if walkErr.JobFuncErr != expectedErr {
		t.Error(`expected  walkErr.JobErr == expectedErr`)
	}
}

func ExampleWalk() {
	// called for each complete job
	each := func(done checksum.Job, err error) error {
		if err != nil {
			return err
		}
		if done.SumString(checksum.MD5) == "e8c078f0e4ad79b16fcb618a3790c2df" {
			fmt.Println(done.SumString(checksum.SHA1))
		}
		return nil
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
