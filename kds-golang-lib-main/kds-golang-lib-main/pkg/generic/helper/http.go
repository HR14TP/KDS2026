package helper

import (
	"bytes"
	"fmt"
	"net/http"
)

func DownloadFileAsString(url string) string {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	newStr := buf.String()
	return newStr
}
