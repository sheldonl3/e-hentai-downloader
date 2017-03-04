package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/PuerkitoBio/goquery"
)

var intervalFlag = flag.Float64("interval", 1, "Interval between each download (sec)")

var errLimitReached = errors.New("You have temporarily reached the limit for how many images you can browse. See http://ehgt.org/g/509.gif for more details")

var httpClient *http.Client

func init() {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}
	httpClient = &http.Client{
		Jar: jar,
	}
}

func main() {
	flag.Parse()
	url0 := flag.Arg(0)
	url1 := ""
	for url0 != url1 {
		fmt.Printf("Scraping %s...", url0)
		imgURL, nextURL, err := scrapeImgAndNext(url0)
		if err != nil {
			fmt.Println(err)
			fmt.Println("Retry...")
			continue
		}
		err = download(imgURL)
		if err != nil {
			fmt.Println(err)
			if err == errLimitReached {
				return
			}
			fmt.Println("Retry...")
		} else {
			url0, url1 = nextURL, url0
			fmt.Println("done")
		}
		if *intervalFlag > 0 {
			fmt.Printf("Waiting for %f seconds...", *intervalFlag)
			time.Sleep(time.Duration(*intervalFlag) * time.Second)
			fmt.Println("OK.")
		}
	}
}

func scrapeImgAndNext(rawurl string) (img string, next string, err error) {
	res, err := httpClient.Get(rawurl)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		return "", "", err
	}
	imgNode := doc.Find("#img")
	if imgNode.Length() == 0 {
		return "", "", fmt.Errorf("Can't find #img node")
	}
	imgURL, ok := imgNode.Attr("src")
	if !ok {
		return "", "", fmt.Errorf("Can't find #img src")
	}
	nextURL, ok := imgNode.Parent().Attr("href")
	if !ok {
		return "", "", fmt.Errorf("Can't find next url")
	}
	return imgURL, nextURL, nil
}

func download(rawurl string) error {
	filename, err := fileNameOf(rawurl)
	if err != nil {
		return err
	}
	if filename == "509.gif" {
		return errLimitReached
	}
	resp, err := http.Get(rawurl)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

var reInPath = regexp.MustCompile("[^/]+$")
var reInQuery = regexp.MustCompile("[^=]+$")

func fileNameOf(rawurl string) (string, error) {
	url, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	file := reInPath.FindString(url.Path)
	if file == "image.php" { // Ocasionally, it returns image.php!
		file = reInQuery.FindString(url.RawQuery)
	}
	if file == "" {
		return "", fmt.Errorf("Filename not found: %s", rawurl)
	}
	return file, nil
}
