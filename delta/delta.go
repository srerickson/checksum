package delta

// A Delta represents changes between two sets of files
type Delta struct {
	// all filenames and corresponding digests
	allNames map[string]*digestPair
	// allDigests in v1 and v2
	allDigests map[string]*digestInfo
}

// digestPair is pair of digests associated with a file.
// if v1 == v2 && v1 == "", the file is not changed
// if v1 != v2, the file is new, removed, or changed
// if v1 == "", file is not in v1
// if v2 == "", file is not in v2
type digestPair struct {
	v1 string // file's digest in v1
	v2 string // file's digest in v2
}

// digestInfo is info stored on each digest in allDigests
type digestInfo struct {
	// paths in v1 but not in v2
	v1Removed []string
	// paths in v2 but not in v1
	v2Added []string
	// number of paths with this digest in v1 and v2
	v1In int
	v2In int
}

// a Fileset maps filnames to digests
type FileSet map[string]string

// New returns a new Delta based on changes between v1 and v2
func New(v1 FileSet, v2 FileSet) *Delta {
	var delta Delta
	delta.allNames = make(map[string]*digestPair)
	delta.allDigests = make(map[string]*digestInfo)
	for f, d := range v1 {
		//f = filepath.Clean(f)
		delta.allNames[f] = &digestPair{v1: d}
	}
	for f, d := range v2 {
		//f = filepath.Clean(f)
		if c := delta.allNames[f]; c != nil {
			c.v2 = d
			continue
		}
		delta.allNames[f] = &digestPair{v2: d}
	}
	for f, digs := range delta.allNames {
		if digs.v1 != "" {
			if delta.allDigests[digs.v1] == nil {
				delta.allDigests[digs.v1] = new(digestInfo)
			}
			delta.allDigests[digs.v1].v1In++
		}
		if digs.v2 != "" {
			if delta.allDigests[digs.v2] == nil {
				delta.allDigests[digs.v2] = new(digestInfo)
			}
			delta.allDigests[digs.v2].v2In++
		}
		if digs.v1 == "" && digs.v2 != "" {
			// f only in v2
			info := delta.allDigests[digs.v2]
			info.v2Added = append(info.v2Added, f)
		} else if digs.v1 != "" && digs.v2 == "" {
			// f only in v1
			info := delta.allDigests[digs.v1]
			info.v1Removed = append(info.v1Removed, f)
		}
	}
	return &delta
}

// returns list of files in v2 not in v1
func (d *Delta) Added() []string {
	var added []string
	for _, cd := range d.allDigests {
		if len(cd.v2Added) > len(cd.v1Removed) {
			added = append(added, cd.v2Added[len(cd.v1Removed):]...)
		}
	}
	return added
}

// Removed returns list of files from v1 removed in v2
func (d *Delta) Removed() []string {
	var rem []string
	for _, cd := range d.allDigests {
		if len(cd.v1Removed) > len(cd.v2Added) {
			rem = append(rem, cd.v1Removed[len(cd.v2Added):]...)
		}
	}
	return rem
}

// Renamed returns two equal length slices of filenames.
// The first slice lists the old names. The second slice
// lists correspodning new names.
func (d *Delta) Renamed() ([]string, []string) {
	var v1, v2 []string
	for _, cd := range d.allDigests {
		var min int
		if len(cd.v1Removed) > len(cd.v2Added) {
			min = len(cd.v2Added)
		} else {
			min = len(cd.v1Removed)
		}
		v1 = append(v1, cd.v1Removed[0:min]...)
		v2 = append(v2, cd.v2Added[0:min]...)
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

// Same returns list of files that are the same
// (unmodified) between v1 and v2
func (d *Delta) Same() []string {
	var same []string
	for f, digs := range d.allNames {
		if digs.v1 == digs.v2 {
			same = append(same, f)
		}
	}
	return same
}

// NewDigests returns a map of digests present in v2 but not v1.
// They map key is the digest and the map value is a list of paths
// from v2
func (d *Delta) NewDigests() map[string][]string {
	digs := make(map[string][]string)
	for d, info := range d.allDigests {
		if info.v1In == 0 && info.v2In > 0 {
			digs[d] = make([]string, 0, info.v2In)
		}
	}
	for f, pair := range d.allNames {
		if _, exists := digs[pair.v2]; exists {
			digs[pair.v2] = append(digs[pair.v2], f)
		}
	}
	return digs
}

// RemovedDigests returns a map of digests present in v1 but not v2.
// They map key is the digest and the map value is a list of paths
// from v1
func (d *Delta) RemovedDigests() map[string][]string {
	digs := make(map[string][]string)
	for d, info := range d.allDigests {
		if info.v2In == 0 && info.v1In > 0 {
			digs[d] = make([]string, 0, info.v1In)
		}
	}
	for f, pair := range d.allNames {
		if _, exists := digs[pair.v1]; exists {
			digs[pair.v1] = append(digs[pair.v1], f)
		}
	}
	return digs
}
