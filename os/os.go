package os

import (
	"errors"
	stdos "os"
	"syscall"

	"github.com/forfd8960/sqlite-go/codes"
)

type LockFile struct {
	fd int
}

func DeleteFile(file string) error {
	return stdos.Remove(file)
}

// FileExists return true if file exists
func FileExists(file string) bool {
	_, err := stdos.Stat(file)
	if err == nil {
		return true
	}

	if errors.Is(err, stdos.ErrNotExist) {
		return false
	}

	return false
}

func OpenReadWrite(file string) (lockFile *LockFile, readOnly bool, errCode *codes.ErrCode) {
	fd, err := syscall.Open(file, syscall.O_RDWR|syscall.O_CREAT, 0644)
	if err != nil {
		// if open read & write fails, open it read-only
		fd, err = syscall.Open(file, syscall.O_RDONLY, 0644)
		if err != nil {
			return nil, false, codes.NewCode(codes.SQLiteCanTOpen, err.Error())
		}

		readOnly = true
	} else {
		readOnly = false
	}

	return &LockFile{fd: fd}, readOnly, codes.Ok()
}

func OpenExclusive(file string, del bool) (lockFile *LockFile, code codes.SQLiteCode) {
	if !FileExists(file) {
		return nil, codes.SQLiteCanTOpen
	}

	fd, err := syscall.Open(file, syscall.O_RDWR|syscall.O_CREAT|syscall.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, codes.SQLiteCanTOpen
	}

	if del {
		syscall.Unlink(file)
	}

	return &LockFile{fd: fd}, codes.SQLiteOk
}
