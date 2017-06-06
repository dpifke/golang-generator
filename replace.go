package generator

import (
	"errors"
	"fmt"
	"os"
)

// doAll executes a slice of function pointers, in LIFO order.
func doAll(funcs []func()) {
	for i := len(funcs) - 1; i >= 0; i-- {
		funcs[i]()
	}
}

// AtomicFileReplace overwrites a series of files, but in a safe manner.  If any
// of the file operations fail, they are all rolled back to their original
// state.
func AtomicFileReplace(src, dest []string) error {
	if len(src) != len(dest) {
		return errors.New("mismatched number of source and destination files")
	}

	commit := make([]func(), 0, len(dest))
	rollback := make([]func(), 0, len(dest)*2)

	// Move any existing destination files out of the way
	for i := range dest {
		_, err := os.Stat(dest[i])
		if err == nil {
			tmpFilename := fmt.Sprintf("%s.%d", dest[i], os.Getpid())
			err = os.Rename(dest[i], tmpFilename)
			if err != nil {
				doAll(rollback)
				return err
			}
			rollback = append(rollback, func() { os.Rename(tmpFilename, dest[i]) })
			commit = append(commit, func() { os.Remove(tmpFilename) })
		}
	}

	// Move new files into place
	for i := range src {
		err := os.Rename(src[i], dest[i])
		if err != nil {
			doAll(rollback)
			return err
		}
		rollback = append(rollback, func() { os.Rename(dest[i], src[i]) })
	}

	doAll(commit)
	return nil
}
