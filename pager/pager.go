package pager

import "os"

type Pager struct {
	dbFile      string
	journal     string
	dbfd        *os.File
	jfd         *os.File
	ckptFd      *os.File
	first, last *PgHdr            // list of free pages
	all         *PgHdr            // list of all pages
	pageHash    map[uint64]*PgHdr // map page number to page
}

type PgHdr struct {
	pager              *Pager // the pager to which this pghdr belongs to
	pgNum              uint64
	prevHash, nextHash *PgHdr
	nRef               int32  // number of users of this page
	prevFree, nextFree *PgHdr // free list of pages where nRef == 0
	prevAll, nextAll   *PgHdr // list of all pages
	inJournal          bool   // true if has been written to journal
	inCkpt             bool   // true if written to the checkpoint journal
	dirty              bool   // true if we need write back to changes
}

type PagerI interface {
	Open(dbFile string, maxPage, nExtra uint64) *Pager
}
