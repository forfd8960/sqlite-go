package pager

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJournalMagic(t *testing.T) {
	fmt.Println("journal magic: ", journalMagicBytes)
}

func TestSetErrCode(t *testing.T) {
	p := &Pager{}
	p.errMask |= int(ErrPagerFull)
	fmt.Println("err mask: ", p.errMask)
	assert.True(t, p.errMask&int(ErrPagerFull) > 0)
}
