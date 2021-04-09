package delta_test

import (
	"sort"
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
	d := delta.New(v1, v2)
	added := d.Added()
	sort.StringSlice(added).Sort()
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
	sort.StringSlice(mods).Sort()
	if len(mods) != 2 || mods[0] != "f3" {
		t.Errorf(`expected 2 modified files but got %d: %s`, len(mods), strings.Join(mods, ", "))
	}
	same := d.Same()
	if len(same) != 1 || same[0] != "h1" {
		t.Errorf(`expected 1 unchanged file but got %d: %s`, len(same), strings.Join(same, ", "))
	}
	newDigs := d.NewDigests()
	if len(newDigs) != 4 {
		t.Errorf(`expected NewDigests() to return 4 digests but got %d`, len(newDigs))
	}
	if len(newDigs["qrs-"]) == 0 || newDigs["qrs-"][0] != "h2" {
		t.Error(`expected NewDigests() to return entry for "qrs-" -> h2, but it didn't`)
	}
	if len(newDigs["abc"]) != 0 {
		t.Error(`expected NewDigests() to not return entry for "abc" but it did`)
	}
	remDigs := d.RemovedDigests()
	if len(remDigs) != 2 || len(remDigs["qrs"]) != 1 || len(remDigs["cde"]) != 1 {
		t.Errorf(`expected RemovedDigests() to return ["qrs","cde"] but got %v`, remDigs)
	}

}
