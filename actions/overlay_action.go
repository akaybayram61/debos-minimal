/*
Overlay Action

Recursive copy of directory or file to target filesystem.

	# Yaml syntax:
	- action: overlay
	  origin: name
	  source: directory
	  destination: directory

Mandatory properties:

- source -- relative path to the directory or file located in path referenced by `origin`.
In case if this property is absent then pure path referenced by 'origin' will be used. Source
can include glob patterns

Optional properties:

- origin -- reference to named file or directory.

- destination -- absolute path in the target rootfs where 'source' will be copied.
All existing files will be overwritten.
If destination isn't set '/' of the rootfs will be used.
*/
package actions

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/akaybayram61/debos-minimal"
)

type OverlayAction struct {
	debos.BaseAction `yaml:",inline"`
	Origin           string // origin of overlay, here the export from other action may be used
	Source           string // external path there overlay is
	Destination      string // path inside of rootfs
}

func (overlay *OverlayAction) Verify(context *debos.DebosContext) error {
	if _, err := debos.RestrictedPath(context.Rootdir, overlay.Destination); err != nil {
		return err
	}
	return nil
}

func findMatches(root string, pattern string) ([]string, error) {
	var matches []string

	// Walk through the directory
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the current path matches the pattern
		matched, err := filepath.Match(pattern, info.Name())
		if err != nil {
			return err
		}

		if matched {
			matches = append(matches, path) // Add the full path to matches
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return matches, nil
}

func listDirectories(root string) ([]string, error) {
	var directories []string

	// Read the directory entries
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	// Iterate through the entries
	for _, entry := range entries {
		if entry.IsDir() { // Check if the entry is a directory
			directories = append(directories, entry.Name()) // Add the directory name to the list
		}
	}

	return directories, nil
}

func isGlob(path string) bool {
	lastChar := path[len(path)-1]
	return lastChar == '*'
}

func getGlobPath(path string) string {
	return path[:len(path)-1]
}

func (overlay *OverlayAction) Run(context *debos.DebosContext) error {
	origin := context.RecipeDir

	//Trying to get a filename from exports first
	if len(overlay.Origin) > 0 {
		var found bool
		if origin, found = context.Origin(overlay.Origin); !found {
			return fmt.Errorf("Origin not found '%s'", overlay.Origin)
		}
	}

	if isGlob(overlay.Source) {
		overlay_sources, err := listDirectories(getGlobPath(overlay.Source))

		if err != nil {
			return err
		}

		log.Printf("Overlay List: %s", overlay_sources)

		for _, val := range overlay_sources {
			sourcedir := path.Join(origin, val)

			destination, err := debos.RestrictedPath(context.Rootdir, overlay.Destination)
			if err != nil {
				return err
			}

			log.Printf("Overlaying %s on %s", sourcedir, destination)

			err = debos.CopyTree(sourcedir, destination)

			if err != nil {
				return err
			}
		}
	} else {
		sourcedir := path.Join(origin, overlay.Source)
		destination, err := debos.RestrictedPath(context.Rootdir, overlay.Destination)
		if err != nil {
			return err
		}

		log.Printf("Overlaying %s on %s", sourcedir, destination)
		return debos.CopyTree(sourcedir, destination)
	}

	return nil
}
