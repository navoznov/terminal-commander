package fs

import (
	"os"
	"path/filepath"
	"strings"
)

// Drive is a place the drive dialog offers to jump to.
type Drive struct {
	Name, Path string
}

// Drives lists "/", the home directory as "~" and the mounted volumes.
func Drives(home string) []Drive {
	return drives("/Volumes", home)
}

// drives lists the entries of volumes by name, skipping hidden ones and the
// link to the boot volume.
func drives(volumes, home string) []Drive {
	ds := []Drive{{"/", "/"}}
	if home != "" {
		ds = append(ds, Drive{"~", home})
	}
	des, _ := os.ReadDir(volumes)
	for _, de := range des {
		if strings.HasPrefix(de.Name(), ".") {
			continue
		}
		p := filepath.Join(volumes, de.Name())
		if t, err := filepath.EvalSymlinks(p); err == nil && t == "/" {
			continue
		}
		ds = append(ds, Drive{de.Name(), p})
	}
	return ds
}
