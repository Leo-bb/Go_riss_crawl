package main

import (
	"Leo-bb/Go_riss_crawl/scrape"
	"os"
	"strings"

	"github.com/labstack/echo"
)

const fileNAME string = "papers.csv"

func main() {
	e := echo.New()
	e.GET("/", handleHome)
	e.POST("/scrape", handleScrape)
	e.Logger.Fatal(e.Start(":1323"))
}

func handleHome(c echo.Context) error {
	return c.File("home.html")
}

func handleScrape(c echo.Context) error {
	defer os.Remove(fileNAME)
	keyword := strings.ToLower(scrape.CleanString(c.FormValue("keyword")))
	// page, _ := strconv.Atoi(c.FormValue("page"))
	page := 2
	scrape.Scrape(keyword, page)
	return c.Attachment(fileNAME, fileNAME)
}
