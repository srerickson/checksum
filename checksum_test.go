package checksum

import (
	"context"
	"crypto/md5"
	"testing"
)

func TestWalk(t *testing.T) {
	newMD5 := md5.New
	jobs, done := Walk(`test/fixture`, newMD5)
	for j := range jobs {
		if j.err != nil {
			t.Error(j.Err())
		}
	}
	err := <-done
	if err != nil {
		t.Fatalf(err.Error())
	}
}
func TestWalkContextCancel(t *testing.T) {
	newMD5 := md5.New
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // <-- cancel the context
	jobs, done := Walk(`test/fixture`, newMD5, WithContext(ctx))
	for range jobs {
		t.Error(`unexpected job in canceled context`)
	}
	err := <-done
	if err == nil {
		t.Error(`expected canceled context error`)
	}
}

func TestPoolErr(t *testing.T) {
	in := make(chan Job)
	go func() {
		defer close(in)
		in <- Job{
			Path:    `nofile`,
			HashNew: md5.New,
		}
	}()
	jobsOut := Pool(in)
	result := <-jobsOut
	if result.Err() == nil {
		t.Error(`expected read error`)
	}
	_, alive := <-jobsOut
	if alive {
		t.Error(`expected closed chan`)
	}
}
