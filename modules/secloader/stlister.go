package secloader

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

func init() {

	resp, err := http.Get(SRC_URL)

	if err != nil {
		log.Fatal("Unable to get HTTP body from %s, %s.\n", SRC_URL, err)
		return
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal("Unable to create goquery.Document object from %s, %s.\n", SRC_URL, err)
		return
	}

	// body, _ := ioutil.ReadAll(resp.Body)
	var regression []*goquery.Selection
	// td headers="view-field-display-title-table-column" class="views-field views-field-field-display-title"
	//     a href="/files/dera/data/financial-statement-data-sets/2021q2.zip"
	doc.Find("td[headers=view-field-display-title-table-column]").Each(func(i int, s *goquery.Selection) {
		regression = append(regression, s)
	})

	for _, r := range regression {

		// dlstr: /files/dera/data/financial-statement-data-sets/2021q2.zip
		dlstr, _ := r.Find("a").Attr("href")
		fanmes := strings.Split(dlstr, "/")
		id1 := strings.Index(fanmes[len(fanmes)-1], "q")
		id2 := strings.Index(fanmes[len(fanmes)-1], ".")
		year, _ := strconv.Atoi(fanmes[len(fanmes)-1][:4])
		quart, _ := strconv.Atoi(fanmes[len(fanmes)-1][id1+1 : id2])
		STObjList = append(STObjList, STObj{
			Name:      fanmes[len(fanmes)-1][:id2],
			Address:   SRC_DOMAIN + dlstr,
			Year:      year,
			Quarterly: quart,
		})
		syncLoadSTTab[fanmes[len(fanmes)-1][:id2]] = sync.Once{}
	}

}

func GetList() []STObj { return STObjList }
