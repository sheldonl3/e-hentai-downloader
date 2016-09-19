package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"code.google.com/p/go.net/html"
	"github.com/PuerkitoBio/goquery"
)

var intervalFlag = flag.Float64("interval", 1, "Interval between each download (sec)")

var ErrLimitReached = errors.New("You have temporarily reached the limit for how many images you can browse. See http://ehgt.org/g/509.gif for more details.")

func main() {
	flag.Parse()
	url0 := flag.Arg(0)
	url1 := ""
	for url0 != url1 {
		fmt.Printf("Scraping %s...", url0)
		imgUrl, nextUrl, err := scrapeImgAndNext(url0)
		if err != nil {
			fmt.Println(err)
			fmt.Println("Retry...")
			continue
		}
		err = download(imgUrl)
		if err != nil {
			fmt.Println(err)
			if err == ErrLimitReached {
				return
			}
			fmt.Println("Retry...")
		} else {
			url0, url1 = nextUrl, url0
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
	doc, err := goquery.NewDocument(rawurl)
	if err != nil {
		return "", "", err
	}
	imgNode := doc.Find("#img").Nodes[0]
	if imgNode == nil {
		return "", "", fmt.Errorf("Can't find #img node")
	}
	imgUrl := getAttr(imgNode, "src")
	if imgUrl == "" {
		return "", "", fmt.Errorf("Can't find #img src")
	}
	nextUrl := getAttr(imgNode.Parent, "href")
	if imgUrl == "" {
		return "", "", fmt.Errorf("Can't find next url")
	}
	return imgUrl, nextUrl, nil
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func download(rawurl string) error {
	filename, err := fileNameOf(rawurl)
	if err != nil {
		return err
	}
	if filename == "509.gif" {
		return ErrLimitReached
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
