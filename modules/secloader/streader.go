package secloader

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

func Update(stobj STObj) {
	// readForm will only excute ONCE in mutex way,
	// other requests will wait until the first call of readForm is finished
	// syncLoadSTTab[stobj.Name].Do(readForm(stobj))
	once := syncLoadSTTab[stobj.Name]
	once.Do(func() { readForm(stobj) })
}

func writeBinary(fpath string, tabobj STTab) {
	f, err := os.Create(fpath)
	if err != nil {
		log.Fatal("Can't open file")
	}
	defer f.Close()
	err = binary.Write(f, binary.LittleEndian, tabobj)
	if err != nil {
		log.Fatal("Write failed")
	}

}

func readBinary(fpath string, pstab *STTab) error {
	f, err := os.Open(fpath)
	if err != nil {
		log.Fatal("Can't open file")
		return err
	}
	err = binary.Read(f, binary.LittleEndian, pstab)
	return err
}

func readForm(stobj STObj) {
	fname := stobj.Name + ".bin"
	var err error

	fpath, err := os.Executable()
	if err != nil {
		log.Fatal("Can't get current directory")
	}

	fpath = path.Join(fpath, "cache", fname)
	_, err = os.Stat(fpath)

	if err != nil && os.IsNotExist(err) {
		// cache file not exist, download and process
		resp, err := http.Get(stobj.Address)
		if err != nil {
			log.Fatal("Failed to download row data at %s.\n", stobj.Address)
			return
		}
		defer resp.Body.Close()

		// download and
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("Failed to download row data at %s.\n", stobj.Address)
			return
		}
		// buff := bufio.NewReader(resp)
		// rdsize, err := io.Copy(buff, resp.Body)

		zobj, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))

		// unpack num.txt, pre.txt, sub.txt, tag.txt
		for _, f := range zobj.File {
			switch strings.ToLower(f.Name[:3]) {
			case "num":
				processNumTxt(stobj, f)
				// default:
				// 	nil
			}
		}

	} else {
		err = readBinary(fpath, &tabobj)
	}
	mapSTTab[fname] = tabobj
}

func processNumTxt(stobj STObj, f *zip.File) {
	zio, err := f.Open()
	if err != nil {
		log.Fatal("Failed to open zip file %s: %s.\n", f.Name, err)
		return
	}
	bio := bufio.NewReader(zio)
	columnline, err := bio.ReadString('\n')
	// tab = adsh	tag	version	coreg	ddate	qtrs	uom	value	footnote
	columns := strings.Split(columnline, "\t")
	var datatable = make(map[string]map[string]interface{})
	for {
		// tline example: 0001640334-21-000798	AccountsPayableAndAccruedLiabilitiesCurrent	us-gaap/2019	20210131	0	USD	10010.0000
		tline, err := bio.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Fatal("Err at reading file %s: %s.\n", f.Name, err)
			}
			break
		}
		var id0 uint64
		var id1, id2, zerov uint
		var tag, version, y4m2d2, currency string
		var tvalue float32
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
		_, ok := datatable[adsh]
		if !ok {
			datatable[adsh]["currency"] = currency
		}
		datatable[adsh][tag] = tvalue

	}

	for adsh, v := range datatable {
		updateDB(StrtoCID(adsh), stobj.Year, stobj.Quarterly, v)
	}

	return
}
