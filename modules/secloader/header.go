package secloader

import (
	"fmt"
	"sync"
	"time"
)

type STObj struct {
	Name      string
	Address   string
	Size      int
	Year      int
	Quarterly int
}

type SingleLine struct {
	Key string
}

type STTab struct {
	// e.g. 2020q2
	Name string
	// adsh -> tag -> value
	datas map[string]map[string]interface{}
}

type CompanyID struct {
	ID0 uint64
	ID1 uint
	ID2 uint
}

type CompanyST struct {
	CID    CompanyID
	Dates  []time.Time
	Tables []AccountTab
}

type AccountTab = map[string]interface{}

const SRC_URL = "https://www.sec.gov/dera/data/financial-statement-data-sets.html"
const SRC_DOMAIN = "www.sec.gov"
const isdebug = true

// read only list, will be initilized at stlister.init
var STObjList []STObj

// maps, file name of statement -> statement's data
// Once is an object that will perform exactly one action
var syncLoadSTTab map[string]sync.Once = make(map[string]sync.Once)
var mapSTTab map[string]STTab = make(map[string]STTab)

func (cid CompanyID) String() string {
	// adsh: 0001477932-21-003366 (10-2-6) unique company id
	return fmt.Sprintf("%010d-%02d-%06d", cid.ID0, cid.ID1, cid.ID2)
}

func StrtoCID(adsh string) CompanyID {
	t := CompanyID{0, 0, 0}
	fmt.Sscanf(adsh, "%d-%d-%d", &t.ID0, &t.ID1, &t.ID2)
	return t
}

func QuartToMonth(quarterly int) int {
	// US fiscal period
	switch quarterly {
	case 1:
		// January, February, and March (Q1)
		return 4
	case 2:
		// April, May, and June (Q2)
		return 7
	case 3:
		// July, August, and September (Q3)
		return 10
	default:
		// October, November, and December (Q4)
		return 1
	}
}
