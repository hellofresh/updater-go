package updater

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Apply performs an update of the current executable (or opts.TargetFile, if set) with the contents of the given
// io.Reader.
//
// Apply performs the following actions to ensure a safe cross-platform update:
//   - Creates a new file, /path/to/.target.new with the TargetMode with the contents of the updated file
//   - Renames /path/to/target to /path/to/.target.old
//   - Renames /path/to/.target.new to /path/to/target
//   - If the final rename is successful, deletes /path/to/.target.old, returns no error. On Windows,
//     the removal of /path/to/target.old always fails, so instead Apply hides the old file instead.
//   - If the final rename fails, attempts to roll back by renaming /path/to/.target.old
//     back to /path/to/target.
//
// If the rollback operation fails, the file system is left in an inconsistent state where there is no new executable
// file and the old executable file could not be be moved to its original location. In this case you should notify the
// user of the bad news and ask them to recover manually. Applications can determine whether the rollback failed by
// calling RollbackError, see the documentation on that function for additional detail.
func Apply(update io.Reader, targetPath string, targetMode os.FileMode) error {
	if targetPath == "" {
		executablePath, executableMode, err := executableInfo()
		if err != nil {
			return err
		}

		targetPath = executablePath
		if targetMode == 0 {
			targetMode = executableMode
		}
	} else if targetMode == 0 {
		targetStats, err := os.Stat(targetPath)
		if err != nil {
			return err
		}
		targetMode = targetStats.Mode()
	}

	// Get the directory the executable exists in.
	updateDir := filepath.Dir(targetPath)
	filename := filepath.Base(targetPath)

	// Define paths to use.
	newPath := filepath.Join(updateDir, fmt.Sprintf(".%s.new", filename))
	oldPath := filepath.Join(updateDir, fmt.Sprintf(".%s.old", filename))

	// Copy the new executable to a new file.
	if err := copyToFile(update, newPath, targetMode); err != nil {
		return err
	}

	// Delete any existing old exec file.
	os.Remove(oldPath)

	// move the existing executable to a new file in the same directory
	if err := os.Rename(targetPath, oldPath); err != nil {
		return err
	}

	// Move the new executable in to become the new program.
	if err := os.Rename(newPath, targetPath); err != nil {
		// Move unsuccessful.
		//
		// The filesystem is now in a bad state. We have successfully moved the existing binary to a new location, but
		// we couldn't move the new binary to take its place. That means there is no file where the current executable
		// binary used to be!
		//
		// Try to rollback by restoring the old binary to its original path.
		rollbackErr := os.Rename(oldPath, targetPath)
		if rollbackErr != nil {
			return RollbackErr{err, rollbackErr}
		}

		return err
	}

	// Remove the old binary.
	if err := os.Remove(oldPath); err != nil {
		// Windows has trouble with removing old binaries, so hide it instead.
		_ = hideFile(oldPath)
	}

	return nil
}

// RollbackErr represents an error occurred during rollback operation.
type RollbackErr struct {
	// error the original error.
	error
	// RollbackErr the error encountered while rolling back.
	RollbackErr error
}

func copyToFile(src io.Reader, dst string, mode os.FileMode) error {
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err = io.Copy(f, src); err != nil {
		return err
	}

	return f.Sync()
}
