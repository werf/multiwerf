package http

import (
	"fmt"
	"io"
	"io/ioutil"
	netHttp "net/http"
	"os"
	"path/filepath"
	"time"
)

func MakeRestAPICall(method string, url string) (content string, err error) {
	var netClient = &netHttp.Client{
		Timeout: time.Second * 30,
	}

	response, err := netClient.Get(url)
	if err != nil {
		return
	} else {
		defer response.Body.Close()
		data, _ := ioutil.ReadAll(response.Body)
		content = string(data)
		return
	}
}

// DownloadLargeFile creates a dstPath/name file and write content form url
// TODO download to tmp file, and copy after successful download. Remove all traces if error.
// TODO Timeouts!
// TODO progress ticks
func DownloadLargeFile(srcUrl string, dstPath string, name string) (err error) {
	//
	err = os.MkdirAll(dstPath, 0755)
	if err != nil {
		return err
	}

	filePath := filepath.Join(dstPath, name)

	// Create the file
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	//fmt.Printf("GET %s into %s %s\n", srcUrl, dstPath, name)
	resp, err := netHttp.Get(srcUrl)
	if err != nil {
		fmt.Printf("get error: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	// Check server response
	//fmt.Printf("status %v\n", resp.Status)
	if resp.StatusCode != netHttp.StatusOK {
		return fmt.Errorf("bad status: %v", resp.Status)
	}

	//fmt.Printf("start copy\n")
	// Stream response body to a file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		//fmt.Printf("Copy error: %v\n", err)
		return err
	}

	//fmt.Printf("Copy written %d bytes\n", written)

	return nil
}
