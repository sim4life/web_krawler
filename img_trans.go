package main

import (
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthonynsimon/bild/imgio"
	"github.com/anthonynsimon/bild/transform"
)

type ImgUrl struct {
	dirName string
	url     string
}

func rotateImg(img image.Image) image.Image {
	fmt.Println("Waiting for jpeg image...")
	return transform.Rotate(img, 270.0, nil)
}

func SaveImg(imgUrlQueue chan *ImgUrl, client http.Client) {
	for imgUrl := range imgUrlQueue {
		resp, err := client.Get(imgUrl.url)
		check(err)
		// defer resp.Body.Close()

		// saveImgToFile(imgUrl, resp)
		img, err := readImage(resp.Body)
		if err != nil {
			log.Println(err.Error())
		}
		resp.Body.Close()

		img = rotateImg(img)
		saveTransformedImg(imgUrl, img)

	}

}

func readImage(reader io.Reader) (image.Image, error) {
	m, f, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("%s (%s)", err.Error(), f)
	}
	return m, nil
}

func getFileName(url string) string {
	i := strings.LastIndex(url, "/")
	if i > -1 {
		return url[i+1:]
	}
	return url
}

func saveTransformedImg(imgUrl *ImgUrl, img image.Image) {
	imgPath := "data" + string(filepath.Separator) + imgUrl.dirName + string(filepath.Separator) + getFileName(imgUrl.url)

	if err := imgio.Save(imgPath, img, imgio.JPEG); err != nil {
		log.Println(err.Error())
	}
}

func saveImgToFile(imgUrl *ImgUrl, resp *http.Response) {
	imgPath := "data" + string(filepath.Separator) + imgUrl.dirName + string(filepath.Separator) + getFileName(imgUrl.url)
	// fmt.Println(imgPath)
	fileHandle, err := os.OpenFile(imgPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0766)
	if err != nil {
		log.Println(err.Error())
	}
	defer fileHandle.Close()
	_, err = io.Copy(fileHandle, resp.Body)
	if err != nil {
		log.Println(err.Error())
	}
}
