package codes

type SQLiteCode int

const (
	SQLiteOk       SQLiteCode = iota // successful
	SQLiteError                      // SQL error or missing database
	SQLiteInternal                   // An internal logic error in SQLite
	SQLitePerm                       // access perm denied
	SQLiteAbort                      // callback routine request an abort
	SQLiteBusy                       // the database file is locked
	SQLiteLocked
	SQLiteNoMem
	SQLiteReadOnly
	SQLiteInterrupt
	SQLiteIOErr
	SQLiteCorrupt
	SQLiteNotFound
	SQLiteFull
	SQLiteCanTOpen
	SQLiteProtocol
	SQLiteEmpty
	SQLiteSchema
	SQLiteTooBig     // too much data for one row of a table
	SQLiteConstraint // abort due to constraint violation
	SQLiteMisMatch   // data type mismatch
	SQLiteMisUse     // library used incorrectly
)

type ErrCode struct {
	code SQLiteCode
	desc string
}

func Ok() *ErrCode {
	return &ErrCode{code: SQLiteOk}
}

func NewCode(code SQLiteCode, desc string) *ErrCode {
	return &ErrCode{code: code, desc: desc}
}
