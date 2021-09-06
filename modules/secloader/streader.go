package secloader

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

var DEFAULT_TEMP string

func init() {
	DEFAULT_TEMP = os.TempDir()
}

func Update(stobj STObj) {
	// readForm will only excute ONCE in mutex way,
	// other requests will wait until the first call of readForm is finished
	// syncLoadSTTab[stobj.Name].Do(readForm(stobj))
	once := syncLoadSTTab[stobj.Name]
	once.Do(func() { readForm(stobj) })
}

func writeBinary(fpath string, pstab *STTab) {
	var temp bytes.Buffer
	var err error
	enc := gob.NewEncoder(&temp)
	err = enc.Encode(*pstab)
	if err != nil {
		log.Fatalf("Failed to encode to binary: %s", err)

	}

	os.MkdirAll(path.Dir(fpath), os.ModePerm)
	err = os.WriteFile(fpath, temp.Bytes(), 0666)
	if err != nil {
		log.Fatalf("Can't create file %s: %s", fpath, err)
	}
}

func readBinary(fpath string, pstab *STTab) error {
	f, err := os.Open(fpath)
	if err != nil {
		log.Fatalf("Can't open file %s: %s", fpath, err)
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	err = dec.Decode(pstab)
	return err
}

func readForm(stobj STObj) {
	fname := stobj.Name + ".bin"
	var err error
	var fpath = path.Join(DEFAULT_TEMP, "cache", fname)
	_, err = os.Stat(fpath)
	var tabobj STTab
	if err != nil && os.IsNotExist(err) {
		// cache file not exist, download and process
		resp, err := http.Get("http://" + stobj.Address)
		if err != nil {
			log.Fatalf("Failed to download row data at %s.\n", stobj.Address)
			return
		}
		defer resp.Body.Close()

		// download and
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Failed to download row data at %s.\n", stobj.Address)
			return
		}
		// buff := bufio.NewReader(resp)
		// rdsize, err := io.Copy(buff, resp.Body)

		zobj, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))

		// unpack num.txt, pre.txt, sub.txt, tag.txt
		for _, f := range zobj.File {
			switch strings.ToLower(f.Name[:3]) {
			case "num":
				processNumTxt(stobj.Name, stobj.Year, stobj.Quarterly, f, &tabobj)
				// default:
				// 	nil
			}
		}
		writeBinary(fpath, &tabobj)
	} else {
		err = readBinary(fpath, &tabobj)
	}
	mapSTTab[fname] = tabobj
}

func processNumTxt(stname string, styear int, stquart int, f *zip.File, pstab *STTab) {
	var err error
	zio, err := f.Open()
	if err != nil {
		log.Fatalf("Failed to open zip file %s: %s.\n", f.Name, err)
		return
	}
	bio := bufio.NewReader(zio)
	// tab = adsh	tag	version	coreg	ddate	qtrs	uom	value	footnote
	_, err = bio.ReadString('\n')
	// columns := strings.Split(columnline, "\t")
	var datatables = make(map[string]AccountTab)
	var id0 uint64
	var id1, id2, zerov uint
	var tag, version, y4m2d2, currency string
	var tvalue float32
	for {
		// tline example: 0001640334-21-000798	AccountsPayableAndAccruedLiabilitiesCurrent	us-gaap/2019	20210131	0	USD	10010.0000
		tline, err := bio.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Fatalf("Err at reading file %s: %s.\n", f.Name, err)
			}
			break
		}

		fmt.Sscanf(tline, "%d-%d-%d\t%s\t%s\t%s\t%d\t%s\t%f",
			&id0, &id1, &id2,
			&tag, &version, &y4m2d2,
			&zerov,
			&currency,
			&tvalue,
		)
		adsh := strings.Split(tline, "\t")[0]
		// datas := strings.Split(tline, "\t")

		// adsh: 0001477932-21-003366 (10-2-6) unique company id
		// idstr := strings.Split(datas[0], "-")
		// id0, err := strconv.Atoi(idstr[0])
		// id1, err := strconv.Atoi(idstr[1])
		// id2, err := strconv.Atoi(idstr[2])

		// tag: string like AccountsPayableAndAccruedLiabilitiesCurrent
		// tag := datas[1]

		// version: string like us-gaap/2019, ignore
		// version := datas[2]

		// string like 20201231
		// year, err := strconv.Atoi(datas[3][0:4])
		// month, err := strconv.Atoi(datas[3][4:6])
		// date, err := strconv.Atoi(datas[3][6:8])

		// string of 0
		// data[4]
		_, ok := datatables[adsh]
		if !ok {
			datatables[adsh] = AccountTab{"Currency": currency}
		}
		datatables[adsh][tag] = tvalue

	}

	for adsh, v := range datatables {
		updateDB(StrtoCID(adsh), styear, stquart, v)
	}

	(*pstab).Name = stname
	(*pstab).Datas = datatables

	return
}
