package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/PuerkitoBio/goquery"
)

type Gallery struct {
	title    string
	numImgs  int
	startURL string
}

func VerifyURL(url string) bool {
	fmt.Printf("Verifying url:-%s...\n", url)
	return Regex.MatchString(url)
}

func replaceAtIndex(str string, replacement string, startInd, endInd int) string {
	return str[:startInd] + replacement + str[startInd+endInd:]
}

func replaceNum(str string, num int) string {
	var strRep string
	strNum := strconv.Itoa(num)
	i := strings.LastIndex(str, strNum)
	if i > -1 {
		strRep = replaceAtIndex(str, "", i, len(strNum))
	}
	if num < 10 {
		i = strings.LastIndex(strRep, "0")
		if i > -1 {
			strRep = replaceAtIndex(strRep, "", i, 1)
		}
	}
	return strRep
}

/**
 * This function parses and returns the uri associated with the HTML anchor
 * <a href="http://www..."...> tag
 * This function assumes that 'href' attribute contains absolute url.
 * It returns "" empty string if it can't find href attribute from the
 * goquery.Selection parameter.
 */
func getUri(sel *goquery.Selection, base string) string {
	if sel != nil {
		str, exists := sel.Attr("href")
		if exists {
			uri, err := url.Parse(str)
			if err != nil {
				return ""
			}
			baseUrl, err := url.Parse(base)
			if err != nil {
				return ""
			}
			uri = baseUrl.ResolveReference(uri)
			return uri.String()
		}
	}
	return ""
}

/**
 * This function parses a value string parameter and returns int value
 * embedded within the string. It returns 0 if it doesn't find any
 * int value in the value string.
 * Example: "some456more" would return 456
 */
func extractLastInt(value string) int {
	var sc scanner.Scanner
	var tok rune
	var newVal, val int64
	var valInt int
	var err error
	var isFound bool

	if len(value) > 0 {
		sc.Init(strings.NewReader(value))
		sc.Mode = scanner.ScanInts

		for tok != scanner.EOF {
			tok = sc.Scan()
			// fmt.Println("At position", sc.Pos(), ":", sc.TokenText())
			val, err = strconv.ParseInt(sc.TokenText(), 10, 32)
			if err == nil {
				isFound = true
				newVal = val
			}
		}
	}

	if isFound {
		valInt = int(newVal)
	}

	return valInt
}

func FetchFinalPage(urlQueue chan string, imgGallQueue chan *Gallery, client http.Client) {
	for uri := range urlQueue {
		fmt.Printf("Getching uri:-%s...\n", uri)
		resp, err := client.Get(uri)
		check(err)
		// defer resp.Body.Close()

		base := resp.Request.URL
		baseUri := base.Scheme + "://" + base.Host

		// respBytes, err := httputil.DumpResponse(resp, true)
		// check(err)

		doc, err := goquery.NewDocumentFromResponse(resp)
		check(err)
		resp.Body.Close()

		fmt.Println("about to find stuff")
		// Find required info within the document
		title := strings.TrimSpace(doc.Find("h1").First().Text())

		var titleInner, imgZeroUrl, href, newHref string
		var maxNum, newNum, num, hrefN, newHrefN int
		// var realA *goquery.Selection
		/*realA = */ doc.Find("h1").First().NextAll().Find("a").Has("img").Each(func(i int, s *goquery.Selection) {
			var isT bool
			titleInner, isT = s.Attr("title")
			if isT && title != "" && strings.Contains(titleInner, title) {
				if imgZeroUrl == "" {
					imgZeroUrl = getUri(s, baseUri)
				}
				href = newHref
				newHref = getUri(s, baseUri)
				// fmt.Printf("[1] imgZeroUrl is:[%d]%s\n", i, imgZeroUrl)
				// fmt.Printf("[1] href:%s-\nnewHref:%s-\n", href, newHref)

				// fmt.Printf("[1] B4 newNum vs maxNum -- num: %d vs %d ---%d\n", newNum, maxNum, num)
				num = newNum
				newNum = fetchMaxNum(href, newHref)
				// fmt.Printf("[1] newNum: %d\n", newNum)
				if num+1 == newNum {
					maxNum = newNum
				}

				// fmt.Printf("[1] A3 maxNum -- num: %d ---%d\n", maxNum, num)
			} else {
				href = newHref
				newHref = getUri(s, baseUri)
				// fmt.Printf("[2] imgZeroUrl is:[%d]%s\n", i, imgZeroUrl)
				// fmt.Printf("[2] href:%s-\nnewHref:%s-\n", href, newHref)

				// fmt.Printf("[2] B4 newNum vs maxNum -- num: %d vs %d ---%d\n", newNum, maxNum, num)
				hrefN = extractLastInt(href)
				newHrefN = extractLastInt(newHref)
				if hrefN+1 == newHrefN && imgZeroUrl == "" && isBaseURLSame(href, newHref) {
					imgZeroUrl = href
				}

				num = newNum
				newNum = fetchMaxNum(href, newHref)
				// fmt.Printf("[2] newNum: %d\n", newNum)
				if num+1 == newNum {
					maxNum = newNum
				}
				// fmt.Printf("[2] A3 maxNum -- num: %d ---%d\n", maxNum, num)
			}
		})

		// outerHtml, _ := goquery.OuterHtml(realA)
		// html, _ := realA.Html()

		fmt.Printf("Title is:%s-\n", title)
		fmt.Printf("titleInner is:%s-\n", titleInner)
		fmt.Printf("maxNum is:%d-\n", maxNum)
		fmt.Printf("\n---images URL is:\n%s\n---\n", imgZeroUrl)
		// fmt.Printf("OuterHtml is:%s-\n", outerHtml)
		// fmt.Printf("html is:%s-\n", html)
		if VerifyURL(imgZeroUrl) {
			gall := &Gallery{title, maxNum, imgZeroUrl}
			imgGallQueue <- gall
		}
		// writeToFile(respBytes)

	}
}

func FetchImages(imgGallQueue chan *Gallery, imgUrlQueue chan *ImgUrl, client http.Client) {
	for gallery := range imgGallQueue {
		fmt.Printf("\nGetching img uri:-%s...\n", gallery.startURL)
		resp, err := client.Get(gallery.startURL)
		check(err)
		// defer resp.Body.Close()

		base := resp.Request.URL
		baseUri := base.Scheme + "://" + base.Host

		fmt.Printf("base is:%s\n", baseUri)

		startNum := extractLastInt(gallery.startURL)
		doc, err := goquery.NewDocumentFromResponse(resp)
		check(err)
		resp.Body.Close()

		var imgSrc string
		// var docA *goquery.Selection
		doc.Find("a img").EachWithBreak(func(i int, s *goquery.Selection) bool {
			var isS bool
			// souterHtml, _ := goquery.OuterHtml(s)
			// shtml, _ := s.Html()
			// fmt.Printf("Html is:%s\n", shtml)
			imgSrc, isS = s.Attr("src")
			// imgAlt, _ := s.Attr("alt")
			fmt.Printf("[%d]:", i)
			// fmt.Printf("imgSrc is:%s\n", imgSrc)
			// fmt.Printf("imgAlt is:%s\n", imgAlt)
			if isS {
				imgNum := extractLastInt(imgSrc)
				fmt.Printf("startNum:%d vs imgNum:%d\n", startNum, imgNum)
				if imgNum == startNum {
					if !strings.Contains(imgSrc, "banner") {
						fmt.Printf("imgSrc is:%s-\n", imgSrc)
						return false
					}
				}
			}
			return true
		})

		gallImgUrls := getImgUrls(gallery.numImgs, imgSrc)
		dirName := createDir(gallery.title)
		fmt.Printf("Dir: %s -- imgs are:%#v\n", dirName, gallImgUrls)
		for _, img := range gallImgUrls {
			imgUrl := &ImgUrl{dirName, img}
			imgUrlQueue <- imgUrl
		}

	}
}

func fetchMaxNum(href, newHref string) int {
	// var emp rune
	var hrefN int
	if newHref != "" {
		hrefN = extractLastInt(href)
		newHrefN := extractLastInt(newHref)
		// fmt.Printf("hrefN:%d- newHrefN:%d\n", hrefN, newHrefN)
		if hrefN+1 == newHrefN {
			//fmt.Printf("\nhrefN:%d- newHrefN:%d \nequals:%t\n", hrefN, newHrefN, (strings.Compare(hrefR, newHrefR) == 0))
			//fmt.Printf("hrefR:\n-%s-\n-%s-\nnewHrefR\n", hrefR, newHrefR)
			if isBaseURLSame(href, newHref) {
				return newHrefN
			}
		}
	}
	return hrefN
}

func createDir(title string) string {
	titleSlc := strings.Split(title, " ")
	var dirName string
	var found bool
	commons := []string{"the", "of", "and", "in", "her", "with", "for"}
	selTitles := make([]string, 0)
	for _, word := range titleSlc {
		if word != "" {
			wordLow := strings.ToLower(word)
			found = false
			for _, str := range commons {
				if wordLow == str {
					found = true
					break
				}
			}
			if !found {
				dirName = dirName + wordLow[:1]
				selTitles = append(selTitles, wordLow)
			}
		}
	}
	if len(dirName) < 4 {
		dirName = ""
		for _, selTit := range selTitles {
			dirName = dirName + selTit[:2]
		}
	}
	if len(dirName) < 4 {
		dirName = selTitles[0][:4]
	}

	err := os.Mkdir("data"+string(filepath.Separator)+dirName, 0766)
	if err != nil {
		log.Println(err.Error())
	}
	return dirName
}

func getCommonUrl(url string) string {
	urlN := extractLastInt(url)
	urlR := replaceNum(url, urlN)
	return urlR
}

func isBaseURLSame(href, newHref string) bool {
	hrefR := getCommonUrl(href)
	newHrefR := getCommonUrl(newHref)
	//fmt.Printf("\nhrefN:%d- newHrefN:%d \nequals:%t\n", hrefN, newHrefN, (strings.Compare(hrefR, newHrefR) == 0))
	//fmt.Printf("hrefR:\n-%s-\n-%s-\nnewHrefR\n", hrefR, newHrefR)
	if hrefR == newHrefR {
		return true
	}
	return false
}

func appendNumUrl(num int, url string) string {
	i := strings.LastIndex(url, ".")
	if i > -1 {
		url = replaceAtIndex(url, strconv.Itoa(num)+".", i, 1)
	}
	return url
}

func getImgUrls(count int, startUrl string) []string {
	url := startUrl
	urls := make([]string, count+1)
	num := extractLastInt(startUrl)
	strNum := strconv.Itoa(num)
	in := strings.LastIndex(startUrl, strNum)
	if in > -1 {
		url = replaceAtIndex(startUrl, "", in, len(strNum))
	}
	commUrl := getCommonUrl(startUrl)

	i := 0
	for ; i <= count && i < 10; i++ {
		urls[i] = appendNumUrl(i, url)
	}
	for ; i <= count; i++ {
		urls[i] = appendNumUrl(i, commUrl)
	}
	return urls
}

func writeToFile(respBytes []byte) {

	// bytes, _ := ioutil.ReadAll(resp.Body)

	// fmt.Println("HTML:\n\n", string(bytes))
	fileHandle, err := os.OpenFile("data/images.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	check(err)
	writer := bufio.NewWriter(fileHandle)
	defer fileHandle.Close()
	fmt.Fprintln(writer, string(respBytes))
	fmt.Fprintln(writer, "--------------**************########**************--------------")
	writer.Flush()

}
