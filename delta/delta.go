package delta

import (
	"path/filepath"
)

type Delta struct {
	// map of all the filenames and their corresponding
	// digests
	allNames map[string]*digests

	// map: digest -> [v1 names..., v2 names ...]
	// only used for filenames that are not in both versions.
	// Used to determine adds, deletions, and renames.
	addDel map[string]*names
}

// A pair of digests associated with a common filename.
// if v1 != v2, the file has changed
type digests struct {
	v1 string
	v2 string
}

// A sequence of filenames associated with a common digest.
// Corresponding entries in v1 and v2 are considered renamed.
// Additional files in v2 are new files.
// Additional files in v1 ar removed files.
type names struct {
	v1 []string
	v2 []string
}

// a Fileset maps filnames to digests
type FileSet map[string]string

func Diff(v1 FileSet, v2 FileSet) *Delta {
	var delta Delta
	delta.allNames = make(map[string]*digests)
	delta.addDel = make(map[string]*names)
	for f, d := range v1 {
		f = filepath.Clean(f)
		delta.allNames[f] = &digests{v1: d}
	}
	for f, d := range v2 {
		f = filepath.Clean(f)
		if c, _ := delta.allNames[f]; c != nil {
			c.v2 = d
			continue
		}
		delta.allNames[f] = &digests{v2: d}
	}
	for f, digs := range delta.allNames {
		if digs.v1 == `` && digs.v2 != `` {
			// f only in v2
			if nams, _ := delta.addDel[digs.v2]; nams != nil {
				nams.v2 = append(nams.v2, f)
			} else {
				delta.addDel[digs.v2] = &names{v2: []string{f}}
			}
		} else if digs.v1 != `` && digs.v2 == `` {
			// f only in v1
			if nams, _ := delta.addDel[digs.v1]; nams != nil {
				nams.v1 = append(nams.v1, f)
			} else {
				delta.addDel[digs.v1] = &names{v1: []string{f}}
			}
		}
	}
	return &delta
}

// returns list of files in v2 not in v1
func (d *Delta) Added() []string {
	added := []string{}
	for _, cd := range d.addDel {
		if len(cd.v2) > len(cd.v1) {
			added = append(added, cd.v2[len(cd.v1):]...)
		}
	}
	return added
}

// Removed returns list of files from v1 removed in v2
func (d *Delta) Removed() []string {
	rem := []string{}
	for _, cd := range d.addDel {
		if len(cd.v1) > len(cd.v2) {
			rem = append(rem, cd.v1[len(cd.v2):]...)
		}
	}
	return rem
}

// Renamed returns two equal length slices of filenames.
// The first slice lists the old names. The second slice
// lists correspodning new names.
func (d *Delta) Renamed() ([]string, []string) {
	var v1, v2 []string
	for _, cd := range d.addDel {
		var min int
		if len(cd.v1) > len(cd.v2) {
			min = len(cd.v2)
		} else {
			min = len(cd.v1)
		}
		v1 = append(v1, cd.v1[0:min]...)
		v2 = append(v2, cd.v2[0:min]...)
	}
	return v1, v2
}

// Modified returns a list of filenames changed
// from v1 to v2.
func (d *Delta) Modified() []string {
	var mods []string
	for f, digs := range d.allNames {
		if digs.v1 == "" || digs.v2 == "" {
			continue
		}
		if digs.v1 != digs.v2 {
			mods = append(mods, f)
		}
	}
	return mods
}
