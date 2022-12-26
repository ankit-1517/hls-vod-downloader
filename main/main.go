package main

import (
	"main/lib/downloader"

	log "github.com/sirupsen/logrus"
)

func downloadVod(url string, outputFolder string, outputFile string) {
	dldr := downloader.NewDownloader()

	err := dldr.DownloadFromMasterUrl(
		url,
		outputFolder,
		outputFile,
	)
	if err != nil {
		log.Errorf("unable to download vod %v from url %v", url, outputFile)
	} else {
		log.Infof("vod %v downloaded successfully!", outputFile)
	}
}

func main() {
	log.SetLevel(log.InfoLevel)

	url := "url"
	outputFolder := "output"
	outputFile := "outputFile.mp4"
	downloadVod(url, outputFolder, outputFile)
}
