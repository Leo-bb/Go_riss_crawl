package scrape

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fedesog/webdriver"
	"golang.org/x/net/html"
)

type extractedPaper struct {
	title  string
	author string
	book   string
	txt    string
	link   string
}

// Scrape riss paper
func Scrape(keyword string, page int) {
	var papers []extractedPaper
	c := make(chan []extractedPaper)

	runtime.GOMAXPROCS(runtime.NumCPU())

	for i := 0; i < page; i++ {
		URL := getBaseURL(keyword, page)
		go getPaperInfo(URL, c)
	}

	for i := 0; i < page; i++ {
		extractedPapers := <-c
		papers = append(papers, extractedPapers...)
	}

	writePapers(papers)
	fmt.Println("Success")
}

func getBaseURL(keyword string, page int) string {
	defer fmt.Println("Get list of paper, now we looking ", strconv.Itoa(page*10), "pages about", keyword)
	urlBeforeKwd := "http://www.riss.or.kr/search/Search.do?isDetailSearch=N&searchGubun=true&viewYn=OP&queryText=&strQuery="
	urlBeforePage := "&exQuery=&exQueryText=&order=%2FDESC&onHanja=false&strSort=RANK&p_year1=&p_year2=&iStartCount="
	urlAfterPageFirst := "&orderBy=&fsearchMethod=search&sflag=1&isFDetailSearch=N&pageNumber=1&resultKeyword="
	urlAfterPageSecond := "&fsearchSort=&fsearchOrder=&limiterList=&limiterListText=&facetList=&facetListText=&fsearchDB=&icate=re_a_kor&colName=re_a_kor&pageScale=10&query="

	URL := urlBeforeKwd + keyword + urlBeforePage + strconv.Itoa(page*10) + urlAfterPageFirst + keyword + urlAfterPageSecond + keyword

	return URL
}

func getPaperInfo(url string, mainC chan<- []extractedPaper) {
	var papers []extractedPaper
	c := make(chan extractedPaper)

	resp, err := http.Get(url)
	checkErr(err)
	checkStatuscode(resp)

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	checkErr(err)

	searchPaperLists := doc.Find(".srchResultListW > ul > li > div.cont")

	searchPaperLists.Each(func(i int, card *goquery.Selection) {
		go extractPaperInfo(card, c)

	})

	for i := 0; i < searchPaperLists.Length(); i++ {
		paper := <-c
		papers = append(papers, paper)
	}

	mainC <- papers

}

func extractPaperInfo(card *goquery.Selection, c chan<- extractedPaper) {
	driverpath, _ := os.Getwd()
	driverpath = driverpath + "/chromedriver"

	paperlinkTag := card.Find("p.title > a ")
	paperAddr, _ := paperlinkTag.Attr("href")
	link := "http://riss.or.kr" + paperAddr

	title, author, book, text := extracting(link, driverpath)

	if len(text) < 1 {
		text = "해당 논문은 요약문이 작성되지 않았습니다."
	}

	if len(title) > 0 {
		c <- extractedPaper{
			title:  CleanString(title),
			author: CleanString(author),
			book:   CleanString(book),
			txt:    text,
			link:   link,
		}
	}

}

func extracting(url string, driverpath string) (string, string, string, string) {
	chromedriver := webdriver.NewChromeDriver(driverpath)

	driverRunErr := chromedriver.Start()
	checkErr(driverRunErr)
	desired := webdriver.Capabilities{"Platform": "Linux"}
	required := webdriver.Capabilities{}

	session, err := chromedriver.NewSession(desired, required)
	checkErr(err)

	openURLerr := session.Url(url)
	checkErr(openURLerr)

	resp, err := session.Source()
	checkErr(err)

	htmlNode, err := html.Parse(strings.NewReader(resp))
	checkErr(err)

	doc := goquery.NewDocumentFromNode(htmlNode)

	title := doc.Find("#soptionview > div > div.thesisInfo > div.thesisInfoTop > h3").Text()
	author := doc.Find("#soptionview > div > div.thesisInfo > div.infoDetail.on > div.infoDetailL > ul > li:nth-child(1) > div > p").Text()
	book := doc.Find("#soptionview > div > div.thesisInfo > div.infoDetail.on > div.infoDetailL > ul > li:nth-child(3) > div > p > a").Text() + doc.Find("#soptionview > div > div.thesisInfo > div.infoDetail.on > div.infoDetailL > ul > li:nth-child(4) > div > p > a").Text()
	text := doc.Find("#soptionview > div > div.innerDiv > div:nth-child(1) > div > div:nth-child(1) > div.text.off > p").Text()

	defer session.Delete()
	defer chromedriver.Stop()

	return title, author, book, text
}

func writePapers(papers []extractedPaper) {
	file, err := os.Create("papers.csv")
	checkErr(err)

	w := csv.NewWriter(file)
	defer w.Flush()

	headers := []string{"Title", "Author", "Book", "Text", "Link"}

	wErr := w.Write(headers)
	checkErr(wErr)

	for _, paper := range papers {
		paperSlice := []string{paper.title, paper.author, paper.book, paper.txt, paper.link}
		jwErr := w.Write(paperSlice)
		checkErr(jwErr)
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func checkStatuscode(resp *http.Response) {
	if resp.StatusCode != 200 {
		log.Fatalln("Request Failed with Status : ", resp.StatusCode)
	}
}

// CleanString clean string
func CleanString(str string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(str)), " ")
}
