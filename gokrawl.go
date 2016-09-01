package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/atotto/clipboard"
)

/* Match for url regexp */
// var Regex = regexp.MustCompile(`/^(https?:\/\/)?([\da-z\.-]+)\.([a-z\.]{2,6})([\/\w \.-]*)*\/?$/`)

var Regex = regexp.MustCompile(`(http|ftp|https):\/\/([\w\-_]+(?:(?:\.[\w\-_]+)+))([\w\-\.,@?^=%&amp;:/~\+#]*[\w\-\@?^=%&amp;/~\+#])?`)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {

	oldText := ""
	fileHandle, err := os.OpenFile("meta/output.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	check(err)
	writer := bufio.NewWriter(fileHandle)
	defer fileHandle.Close()

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := http.Client{Transport: transport}

	urlQueue := make(chan string)
	defer close(urlQueue)
	imgGallQueue := make(chan *Gallery)
	defer close(imgGallQueue)
	imgUrlQueue := make(chan *ImgUrl)
	defer close(imgUrlQueue)
	go FetchFinalPage(urlQueue, imgGallQueue, client)
	go FetchImages(imgGallQueue, imgUrlQueue, client)
	go SaveImg(imgUrlQueue, client)
	for {
		url, _ := clipboard.ReadAll()
		if url != oldText {
			// fmt.Println(text)
			oldText = url
			if VerifyURL(url) {
				fmt.Printf("Fetching url:-%s...\n", url)
				urlQueue <- url

				// writer.WriteString("Writer_Write\n")
				fmt.Fprintln(writer, url)
				writer.Flush()
			} else {
				fmt.Printf("Not a url:-%s-\n", url)
			}
		}
	}
}
