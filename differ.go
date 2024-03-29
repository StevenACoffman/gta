package gta

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Differ implements a single method, used to note code diffs
// and the dirs they occur.
type Differ interface {
	// Diff returns a set of absolute pathed directories
	// that have files that have been modified.
	Diff() (map[string]bool, error)
}

// We check to make sure Git implements the Differ interface.
var _ Differ = &Git{}

// Git implements the Differ interface using a git version control method.
type Git struct {
	UseMergeCommit bool
}

// Diff returns a set of changed directories.
func (g *Git) Diff() (map[string]bool, error) {
	// We get the root of the repository to build our full path.
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, err
	}
	root := strings.TrimSpace(string(out))

	parent1 := "origin/master"
	parent2 := "HEAD"
	if g.UseMergeCommit {
		out, err := exec.Command("git", "log", "-1", "--merges", "--pretty=format:%p").Output()
		if err != nil {
			return nil, err
		}
		parents := strings.TrimSpace(string(out))
		parentSplit := strings.Split(parents, " ")
		if len(parentSplit) != 2 {
			return nil, fmt.Errorf("could not discover parent merge commits")
		}
		parent1 = parentSplit[0]
		parent2 = parentSplit[1]
	}

	cmd := exec.Command("git", "diff", fmt.Sprintf("%s...%s", parent1, parent2), "--name-only")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	dirs, err := diffFileDirectories(root, stdout)
	if err != nil {
		return nil, err
	}

	existsDirs := map[string]bool{}
	for dir := range dirs {
		if exists(dir) {
			existsDirs[dir] = false
		}
	}

	return existsDirs, cmd.Wait()
}

func diffFileDirectories(root string, r io.Reader) (map[string]bool, error) {
	dirs := map[string]bool{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		filename := scanner.Text()

		// We build our full absolute file path.
		full, err := filepath.Abs(filepath.Join(root, filename))
		if err != nil {
			return nil, err
		}
		dirs[filepath.Dir(full)] = false
	}

	return dirs, scanner.Err()
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
