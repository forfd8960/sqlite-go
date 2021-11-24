package pager

import (
	"github.com/forfd8960/sqlite-go/codes"
	"github.com/forfd8960/sqlite-go/os"
)

const (
	SQLITE_PAGE_SIZE = 1024
	SQLITE_MAX_PAGE  = 1073741823
	N_PG_HASH        = 2003
)

type PagerState int

const (
	SQLITE_UNLOCK PagerState = iota
	SQLITE_READLOCK
	SQLITE_WRITELOCK
)

type PagerErrCode int

const (
	ErrPagerFull    PagerErrCode = 0x01 // a write failed
	ErrPagerMEM     PagerErrCode = 0x02 // mem error
	ErrPagerLock    PagerErrCode = 0x04 // error in the locking protocol
	ErrPagerCorrupt PagerErrCode = 0x08 // database or journal corruption
	ErrPagerDisk    PagerErrCode = 0x10 // gerneral dist I/O error
)

var (
	journalMagicBytes = []byte{0xd9, 0xd5, 0x05, 0xf9, 0x20, 0xa1, 0x63, 0xd4}
	pageHash          = func(pn uint) uint { return pn % N_PG_HASH }
)

type Pager struct {
	dbFile              string
	journal             string
	dbfd                *os.LockFile
	jfd                 *os.LockFile
	ckptFd              *os.LockFile
	dbSize, origDBSize  int64
	ckptSize, ckptJSize int64
	nExtra              int64
	destructorFn        func()
	nPage               int64      // total number of in-memory pages
	nRef                int64      // number of in-memory pages with PgHdr.nRef > 0
	maxPage             int64      // max number of pages to hold in cache
	nHit, nMiss, nOvfl  int64      // cache hits, missing, lru overflows
	journalOpen         bool       // true if journal fd is valid
	ckptOpen            bool       // true if the checkpoint journal is open
	ckptInUse           bool       // true if we are in a checkpoint
	noSync              bool       // do not sync the journal if true
	state               PagerState // SQLITE_UNLOCK, _READLOCK, _WRITELOCK
	errMask             int
	tempFile            bool   // dbFile is a temp file
	readOnly            bool   // true for a read-only database
	needSync            bool   // true if an fsync() is need on the journal
	dirtyFile           bool   // true if database file has changed in any way
	aInJournal          *uint8 // one bit for each page in the database
	aInCkpt             *uint8 // One bit for each page in the database

	first, last *PgHdr            // list of free pages
	all         *PgHdr            // list of all pages
	pageHash    map[uint64]*PgHdr // map page number to page
}

type PgHdr struct {
	pager              *Pager // the pager to which this pghdr belongs to
	pgNum              uint
	prevHash, nextHash *PgHdr
	nRef               int32  // number of users of this page
	prevFree, nextFree *PgHdr // free list of pages where nRef == 0
	prevAll, nextAll   *PgHdr // list of all pages
	inJournal          bool   // true if has been written to journal
	inCkpt             bool   // true if written to the checkpoint journal
	dirty              bool   // true if we need write back to changes
}

type PageRecord struct {
	pgNum  uint                   // the page number
	pgData [SQLITE_PAGE_SIZE]byte // original data for page num
}

type PagerI interface {
}

func NewPager(db string, maxPage int64, nExtra int64) (*Pager, codes.SQLiteCode) {
	pager := &Pager{}

	var rc codes.SQLiteCode
	if db != "" {
		pager.dbfd, pager.readOnly, rc = os.OpenReadWrite(db)
	} else {
		//todo: open temp file
	}

	if rc != codes.SQLiteOk {
		return nil, codes.SQLiteCanTOpen
	}

	pager.dbFile = db
	pager.journal = db + "-journal"
	pager.dbSize = -1
	pager.maxPage = 10
	if maxPage > 5 {
		pager.maxPage = maxPage
	}
	pager.state = SQLITE_UNLOCK
	pager.nExtra = nExtra
	pager.pageHash = make(map[uint64]*PgHdr, pager.maxPage)
	return pager, codes.SQLiteOk
}

func (p *Pager) SetDestructor(destructorFn func()) {
	p.destructorFn = destructorFn
}

func (p *Pager) PageCount() int64 {
	if p.dbSize >= 0 {
		return p.dbSize
	}

	size, rc := p.dbfd.Size()
	if rc != codes.SQLiteOk {
		p.errMask |= int(ErrPagerDisk)
		return -1
	}

	pageCount := size / SQLITE_PAGE_SIZE
	if p.state != SQLITE_UNLOCK {
		p.dbSize = pageCount
	}

	return pageCount
}

func (p *Pager) Close() codes.SQLiteCode {
	switch p.state {
	case SQLITE_WRITELOCK:
		//todo: add call of pager_rollback
	case SQLITE_READLOCK:
		p.dbfd.UnLock()
	default:
	}

	p.all = nil
	p.dbfd.Close()
	return codes.SQLiteOk
}

func (pgHdr *PgHdr) PageNumber() uint {
	return pgHdr.pgNum
}

// PageRef Increase the ref count for a page
func (pgHdr *PgHdr) PageRef() {
	pageRef(pgHdr)
}

func pageRef(pgHdr *PgHdr) {
	if pgHdr.nRef == 0 { // if this page has zero ref count, remove it from free list
		if pgHdr.prevFree != nil {
			pgHdr.prevFree.nextFree = pgHdr.nextFree
		} else {
			pgHdr.pager.first = pgHdr.nextFree
		}

		if pgHdr.nextFree != nil {
			pgHdr.nextFree.prevFree = pgHdr.prevFree
		} else {
			pgHdr.pager.last = pgHdr.prevFree
		}

		pgHdr.pager.nRef++
	}
	pgHdr.nRef++
}
