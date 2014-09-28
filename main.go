package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"code.google.com/p/go.net/html"
	"github.com/PuerkitoBio/goquery"
)

func main() {
	url0 := os.Args[1]
	url1 := ""
	for url0 != url1 {
		fmt.Printf("Scraping %s...", url0)
		imgUrl, nextUrl, err := scrapeImgAndNext(url0)
		if err != nil {
			fmt.Println(err)
			log.Fatal(err)
		}
		err = download(imgUrl)
		if err != nil {
			fmt.Println(err)
			log.Fatal(err)
		}
		url0, url1 = nextUrl, url0
		fmt.Println("done")
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

var reFileName = regexp.MustCompile("[^/]+$")

func fileNameOf(rawurl string) (string, error) {
	url, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	file := reFileName.FindString(url.Path)
	if file == "" {
		return "", fmt.Errorf("Filename not found: %s", rawurl)
	}
	return file, nil
}
