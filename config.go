package checksum

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"runtime"
)

const (
	MD5    = `md5`
	SHA1   = `sha1`
	SHA512 = `sha512`
	SHA256 = `sha256`
	//BLAKE2B512 = `blake2b-512`
)

// Config is a common configuration object
// used by Walk(), NewPipe(), and Add().
type Config struct {
	numGos int // number of goroutines in pool
	ctx    context.Context
	algs   map[string]Alg
}

func defaultConfig() Config {
	return Config{
		numGos: runtime.GOMAXPROCS(0),
		ctx:    context.Background(),
	}
}

// WithGos is used set the number of goroutines used by a Pipe.
// Used as an optional argument for NewPipe().
func WithGos(n int) func(*Config) {
	return func(c *Config) {
		if n < 1 {
			n = 1
		}
		c.numGos = n
	}
}

// WithCtx sets a context for Walk() and NewPipe().
func WithCtx(ctx context.Context) func(*Config) {
	return func(c *Config) {
		c.ctx = ctx
	}
}

// WithAlg adds the named algorith to Walk() and NewPipe().
func WithAlg(name string, alg Alg) func(*Config) {
	return func(c *Config) {
		if c.algs == nil {
			c.algs = make(map[string]Alg)
		}
		c.algs[name] = alg
	}
}

// WithMD5 adds the md5 algorith to Walk() and NewPipe().
func WithMD5() func(*Config) {
	return func(c *Config) {
		WithAlg(MD5, md5.New)(c)
	}
}

// WithSHA1 adds the sha1 algorith to Walk() and NewPipe().
func WithSHA1() func(*Config) {
	return func(c *Config) {
		WithAlg(SHA1, sha1.New)(c)
	}
}

// WithSHA256 adds the sha256 algorith to Walk() and NewPipe().
func WithSHA256() func(*Config) {
	return func(c *Config) {
		WithAlg(SHA256, sha256.New)(c)
	}
}

// WithSHA512 adds the sha512 algorith to Walk() and NewPipe().
func WithSHA512() func(*Config) {
	return func(c *Config) {
		WithAlg(SHA512, sha512.New)(c)
	}
}
