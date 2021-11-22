package os

import (
	"errors"
	"io"
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

func (lf *LockFile) Read(buf []byte, amt int) codes.SQLiteCode {
	n, err := syscall.Read(lf.fd, buf)
	if err != nil {
		return codes.SQLiteIOErr
	}

	if n != amt {
		return codes.SQLiteIOErr
	}

	return codes.SQLiteOk
}

func (lf *LockFile) Write(data []byte) codes.SQLiteCode {
	wn, err := syscall.Write(lf.fd, data)
	if err != nil || wn < len(data) {
		return codes.SQLiteFull
	}

	return codes.SQLiteOk
}

func (lf *LockFile) Seek(offset int64) codes.SQLiteCode {
	_, err := syscall.Seek(lf.fd, offset, io.SeekStart)
	if err != nil {
		return codes.SQLiteIOErr
	}

	return codes.SQLiteOk
}

// Sync make sure all writea to a particular file are commited to disk
func (lf *LockFile) Sync() codes.SQLiteCode {
	if err := syscall.Fsync(lf.fd); err != nil {
		return codes.SQLiteIOErr
	}

	return codes.SQLiteOk
}

// Truncate an poen file to a specified size
func (lf *LockFile) Truncate(nByte int64) codes.SQLiteCode {
	if err := syscall.Ftruncate(lf.fd, nByte); err != nil {
		return codes.SQLiteIOErr
	}

	return codes.SQLiteOk
}

// Size file size in bytes
func (lf *LockFile) Size() (size int64, code codes.SQLiteCode) {
	stat := &syscall.Stat_t{}
	if err := syscall.Fstat(lf.fd, stat); err != nil {
		return -1, codes.SQLiteIOErr
	}

	return stat.Size, codes.SQLiteOk
}

func (lf *LockFile) Close() codes.SQLiteCode {
	err := syscall.Close(lf.fd)
	if err != nil {
		return codes.SQLiteIOErr
	}

	return codes.SQLiteOk
}
