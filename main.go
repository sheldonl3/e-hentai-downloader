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
		return url.Parse("http://127.0.0.1:58591")
	}
	transport := &http.Transport{Proxy: proxy}

	httpClient = &http.Client{
		Jar:       jar,
		Transport: transport,
	}
}

/*从画集详情页找到第一个图片url，开始遍历全部url，并开启gorutinue进行下载*/
func main() {
	flag.Parse()
	url, title, err := getImgurlandtitle(flag.Arg(0))
	if err != nil {
		fmt.Println(err)
		return
	}
	err = mkdir(title)
	if err != nil {
		fmt.Println(err)
		return
	}
	url1 := ""
	for url != url1 {
		fmt.Printf("Scraping %s...\n", url)
		imgURL, nextURL, err := scrapeImgAndNext(url)
		if err != nil {
			fmt.Println(err)
			fmt.Printf("cant scrapeImg %s\n", url)
			break
		}
		wait.Add(1)
		go download_pic(imgURL)
		url, url1 = nextURL, url
	}
	wait.Wait()
}

/*从画集详情页获取1st图片页url和题目*/
func getImgurlandtitle(rawurl string) (string, string, error) {
	res, err := httpClient.Get(rawurl)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(res)
	var url string
	var title string

	if err != nil {
		return "", "", err
	}
	url, _ = doc.Find(".gdtm").Find("a").Attr("href")
	title = doc.Find("#gn").Text()
	fmt.Printf("url:%s\ntitle:%s\n", url, title)
	return url, title, nil
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

/*解析某一张图片详情的url，找出图片的源url和下一张的url*/
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

/*下载图片的携程，默认重试一次*/
func download_pic(imgURL string) {
	defer wait.Done()
	retry_time := 1
	for {
		err := download(imgURL)
		if err != nil {
			fmt.Println(err)
			if err == errLimitReached {
				break
			}
			if retry_time != 0 {
				retry_time--
				fmt.Println("Retry...")
			} else {
				fmt.Printf("\033[1;31;40m%s\033[0m\n", "error")
				fmt.Printf("cant download %s\n", imgURL)
				break
			}
		} else {
			break
		}
	}
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
	return nil
}

var reInPath = regexp.MustCompile("[^/]+$")
var reInQuery = regexp.MustCompile("[^=]+$")

/*从url获取图片的文件名*/
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
