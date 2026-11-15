package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/ailequal/ffcc-gci/internal/gci"
	"github.com/ailequal/ffcc-gci/internal/save"
)

// now is the clock used to name backups. Tests replace it.
var now = time.Now

// backupTimeFormat is the timestamp put in a backup's file name.
const backupTimeFormat = "20060102-150405"

// saveFile is a PAL save file read from disk.
type saveFile struct {
	path string      // resolved path of the file
	raw  []byte      // the file's bytes as read
	mode fs.FileMode // the file's permissions
	gci  *gci.File
	data *save.Data
}

// openSave reads and validates the save file at path.
func openSave(path string) (*saveFile, error) {
	// Resolve symbolic links so that an in-place edit replaces the real
	// file and not the link.
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(resolved)
	if err != nil {
		return nil, err
	}

	file, err := gci.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := save.CheckIdentity(file.GameCode(), file.MakerCode(), file.FileName()); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	data, err := save.Parse(file.Data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return &saveFile{path: resolved, raw: raw, mode: info.Mode().Perm(), gci: file, data: data}, nil
}

// withData returns the bytes of the file with its save data replaced.
// The GCI header is kept as it was.
func (f *saveFile) withData(data *save.Data) []byte {
	out := gci.File{Header: f.gci.Header, Data: data.Bytes()}
	return out.Bytes()
}

// writeInPlace replaces the file with content. Before that it saves the
// original bytes to a backup next to the file and returns the backup's
// path.
//
// The backup does not end in ".gci": Dolphin loads every .gci file in a
// memory card folder and rejects a second one with the same internal
// file name (GCMemcardDirectory::LoadGCI).
func (f *saveFile) writeInPlace(content []byte) (backup string, err error) {
	current, err := os.ReadFile(f.path)
	if err != nil {
		return "", err
	}
	if !bytes.Equal(current, f.raw) {
		return "", fmt.Errorf("%s changed on disk while it was being edited; is Dolphin running? Nothing was written", f.path)
	}

	backup = f.path + "." + now().Format(backupTimeFormat) + ".bak"
	if err := writeNew(backup, f.raw, f.mode); err != nil {
		return "", fmt.Errorf("could not write the backup, so the save was left untouched: %w", err)
	}
	if err := replaceFile(f.path, content, f.mode); err != nil {
		return backup, err
	}
	return backup, nil
}

// writeNew writes content to a file that must not exist yet.
func writeNew(path string, content []byte, mode fs.FileMode) (err error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.Remove(path)
		}
	}()

	if err := writeAndClose(file, content); err != nil {
		return err
	}
	return checkWritten(path, content)
}

// replaceFile atomically replaces the file at path with content: it
// writes a temporary file in the same directory and renames it over the
// original, so the save is never left half written.
func replaceFile(path string, content []byte, mode fs.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ffcc-gci-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no effect once the rename has happened

	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := writeAndClose(tmp, content); err != nil {
		return err
	}
	if err := checkWritten(tmp.Name(), content); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func writeAndClose(file *os.File, content []byte) error {
	_, err := file.Write(content)
	if err == nil {
		err = file.Sync()
	}
	return errors.Join(err, file.Close())
}

// checkWritten reads a file back and compares it to what was written.
func checkWritten(path string, content []byte) error {
	got, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(got, content) {
		return fmt.Errorf("%s does not read back as written", path)
	}
	return nil
}
