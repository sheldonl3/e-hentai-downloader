package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/publicsuffix"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"sync"
)

var errLimitReached = errors.New("You have temporarily reached the limit for how many images you can browse. See http://ehgt.org/g/509.gif for more details")

var httpClient *http.Client

var path string

var wait sync.WaitGroup

func init() {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}

	//设置代理
	proxy := func(_ *http.Request) (*url.URL, error) {
		return url.Parse("http://127.0.0.1:51837")
	}
	transport := &http.Transport{Proxy: proxy}

	httpClient = &http.Client{
		Jar:       jar,
		Transport: transport,
	}
}

func main() {
	flag.Parse()
	url0 := flag.Arg(0)
	//getImgurl(url0)
	url1 := ""
	first_url := true
	for url0 != url1 {
		fmt.Printf("Scraping %s...", url0)
		imgURL, nextURL, err := scrapeImgAndNext(url0, &first_url)
		if err != nil {
			if first_url == true {
				fmt.Println(err)
				break
			}
			fmt.Println(err)
			fmt.Println("Retry...")
			continue
		}
		wait.Add(1)
		go download_pic(imgURL)
		url0, url1 = nextURL, url0
	}
	wait.Wait()
}

//func getImgurl(rawurl string) (img string, err error) {
//	res, err := httpClient.Get(rawurl)
//	if err != nil {
//		return "", err
//	}
//	defer res.Body.Close()
//
//	doc, err := goquery.NewDocumentFromResponse(res)
//	if err != nil {
//		return "", err
//	}
//	doc.Find(".gdtm").Each(func(i int, selection *goquery.Selection) {
//		url,_:=selection.Find("a").Attr("href")
//		fmt.Printf("%d - url:%s\n",i,url)
//	})
//
//
//	return "", nil
//}

func scrapeImgAndNext(rawurl string, first_url *bool) (img string, next string, err error) {
	res, err := httpClient.Get(rawurl)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		return "", "", err
	}

	//创建文件夹
	if *first_url == true {
		title := doc.Find("title")
		if title.Length() == 0 {
			return "", "", fmt.Errorf("Can't find #title node")
		}
		err := mkdir(title.Text())
		if err != nil {
			return "", "", err
		}
		fmt.Println("data will be save in " + title.Text())
	}

	imgNode := doc.Find("#img")
	if imgNode.Length() == 0 {
		*first_url = false
		return "", "", fmt.Errorf("Can't find #img node")
	}
	imgURL, ok := imgNode.Attr("src")
	if !ok {
		*first_url = false
		return "", "", fmt.Errorf("Can't find #img src")
	}
	nextURL, ok := imgNode.Parent().Attr("href")
	if !ok {
		*first_url = false
		return "", "", fmt.Errorf("Can't find next url")
	}
	*first_url = false
	return imgURL, nextURL, nil
}

func mkdir(title string) (error error) {
	for i, chr := range title { //防止出现'/'导致文件夹生成失败
		if chr == 47 {
			title = title[i:] + "\\" + title[i+1:]
		}
	}
	path = "./" + title
	_, err := os.Stat(path)
	if err == nil {
		return fmt.Errorf("dir has already exist")
	}
	if os.IsNotExist(err) {
		fmt.Println("not dir")
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			return fmt.Errorf("mkdir failed!\n[%v]\n", err)
		} else {
			return nil
		}
	}
	return fmt.Errorf("dont know " + title + "exits")
}

func download_pic(imgURL string) {
	for {
		err := download(imgURL)
		if err != nil {
			fmt.Println(err)
			if err == errLimitReached {
				break
			}
			fmt.Println("Retry...")
		} else {
			break
		}
	}
	fmt.Println("done")
	return
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
	pic_path := path + "/" + filename
	fmt.Println("gen " + pic_path)
	file, err := os.OpenFile(pic_path, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	wait.Done()
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
