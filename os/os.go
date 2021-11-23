package os

import (
	"errors"
	"io"
	stdos "os"
	"sync"
	"syscall"

	"github.com/forfd8960/sqlite-go/codes"
)

type LockFile struct {
	fd int

	mu     sync.Mutex
	locked bool
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

func OpenReadWrite(file string) (lockFile *LockFile, readOnly bool, errCode codes.SQLiteCode) {
	fd, err := syscall.Open(file, syscall.O_RDWR|syscall.O_CREAT, 0644)
	if err != nil {
		// if open read & write fails, open it read-only
		fd, err = syscall.Open(file, syscall.O_RDONLY, 0644)
		if err != nil {
			return nil, false, codes.SQLiteCanTOpen
		}

		readOnly = true
	} else {
		readOnly = false
	}

	return &LockFile{fd: fd}, readOnly, codes.SQLiteOk
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

func (lf *LockFile) ReadLock() codes.SQLiteCode {
	lf.mu.Lock()
	defer lf.mu.Unlock()

	// locking in shared and non-blocking
	if err := syscall.FcntlFlock(uintptr(lf.fd), syscall.F_SETLK, &syscall.Flock_t{
		Type:   syscall.F_RDLCK,
		Whence: io.SeekStart,
		Start:  0,
		Len:    0,
	}); err != nil {
		return codes.SQLiteBusy
	}

	lf.locked = true
	return codes.SQLiteOk
}

func (lf *LockFile) WriteLock() codes.SQLiteCode {
	lf.mu.Lock()
	defer lf.mu.Unlock()

	if err := syscall.FcntlFlock(uintptr(lf.fd), syscall.F_SETLK, &syscall.Flock_t{
		Type:   syscall.F_WRLCK,
		Whence: int16(io.SeekStart),
		Start:  0,
		Len:    0,
	}); err != nil {
		return codes.SQLiteBusy
	}

	lf.locked = true
	return codes.SQLiteOk
}

func (lf *LockFile) UnLock() codes.SQLiteCode {
	lf.mu.Lock()
	defer lf.mu.Unlock()
	locked := lf.locked

	if !locked {
		return codes.SQLiteOk
	}

	flock := &syscall.Flock_t{
		Type:   syscall.F_UNLCK,
		Whence: io.SeekStart,
		Start:  0,
		Len:    0,
	}
	if err := syscall.FcntlFlock(uintptr(lf.fd), syscall.F_SETLK, flock); err != nil {
		return codes.SQLiteBusy
	}

	lf.locked = false
	return codes.SQLiteOk
}
