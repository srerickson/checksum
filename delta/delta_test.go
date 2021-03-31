package delta_test

import (
	"strings"
	"testing"

	"github.com/srerickson/checksum/delta"
)

func TestDelta(t *testing.T) {
	v1 := delta.FileSet{
		"f1": "abc",
		"f2": "cde",
		"f3": "hij",
		"h1": "hij",
		"h2": "qrs",
	}
	v2 := delta.FileSet{
		"f1-": "abc", // renamed
		// "f2": "cde", // removed
		"f3": "hij-", // modified
		"f4": "xyz",  // new file
		"f5": "abcd", // new file
		"h1": "hij",  // no change
		"h2": "qrs-", // modified
	}
	d := delta.Diff(v1, v2)
	added := d.Added()
	if len(added) != 2 {
		t.Error(`expected 2 additions`)
	}
	if len(added) != 2 || added[0] != "f4" || added[1] != "f5" {
		t.Error(`expected 2 additions called f4 and f5`)
	}
	rem := d.Removed()
	if len(rem) != 1 || rem[0] != "f2" {
		t.Error(`expected 1 removal called f2`)
	}
	old, new := d.Renamed()
	if len(old) != len(new) {
		t.Error(`expected same number of old names and new names from Renamed()`)
	}
	if len(old) != len(new) || len(old) != 1 || new[0] != "f1-" {
		t.Error(`expected 1 renamed file called "f1-"`)
	}
	mods := d.Modified()
	if len(mods) != 2 || mods[1] != "h2" {
		t.Errorf(`expected 2 modified files but got %d: %s`, len(mods), strings.Join(mods, ", "))
	}
}
