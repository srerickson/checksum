package checksum

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"testing"
)

var validSums = map[string]string{
	"test/fixture/folder1/folder2/sculpture-stone-face-head-888027.jpg": "e8c078f0e4ad79b16fcb618a3790c2df",
	"test/fixture/folder1/folder2/file2.txt":                            "d41d8cd98f00b204e9800998ecf8427e",
	"test/fixture/folder1/file.txt":                                     "d41d8cd98f00b204e9800998ecf8427e",
	"test/fixture/hello.csv":                                            "9d02fa6e9dd9f38327f7b213daa28be6",
}

func TestWalk(t *testing.T) {
	newMD5 := md5.New
	pipe := NewPipe()
	err := pipe.Walk(`test/fixture`, newMD5)
	for j := range pipe.Out() {
		if j.Err != nil {
			t.Error(j.Err)
		}
	}
	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestContextCancel(t *testing.T) {
	errs := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	pipe := NewPipe(WithContext(ctx))
	go func() {
		defer pipe.Close()
		pipe.Add(Job{Path: "nofile1", HashNew: md5.New})
		cancel() // <-- cancel the context
		errs <- pipe.Add(Job{Path: "nofile2", HashNew: md5.New})
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
	pipe := NewPipe(WithGoNum(2))
	go func() {
		defer pipe.Close()
		pipe.Add(Job{
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
	pipe := NewPipe()
	go func() {
		defer pipe.Close()
		for path, sum := range validSums {
			sumBytes, _ := hex.DecodeString(sum)
			pipe.Add(Job{Path: path, Valid: sumBytes, HashNew: md5.New})
		}
	}()
	numResults := 0
	for j := range pipe.Out() {
		numResults++
		if !j.IsValid() {
			t.Error(`expected valid result`)
		}
	}
	if numResults != len(validSums) {
		t.Errorf("expected %d results, got %d", len(validSums), numResults)
	}

}
