package main

// TODO: add logger

import (
	"fmt"
	"main/lib/downloader"
)

func main() {
	dldr := downloader.NewDownloader()

	err := dldr.DownloadFromMasterUrl(
		"url",
		"output",
		"videoname.mp4",
	)
	if err != nil {
		fmt.Printf("error occured downloading vod: %v", err.Error())
	} else {
		fmt.Println("vod downloaded successfully!")
	}
}
