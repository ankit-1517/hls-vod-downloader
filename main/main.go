package main

import (
	"main/lib/downloader"

	log "github.com/sirupsen/logrus"
)

func downloadVod(dldr *downloader.Downloader, url string, outputFolder string, outputFile string) {
	err := dldr.DownloadVod(
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

func downloadVodsFromJson(inputFile string, outputFolder string) {
	jsonData, err := parseInputJson(inputFile, outputFolder)
	if err != nil {
		return
	}

	dldr := downloader.NewDownloader()
	for _, reqData := range jsonData {
		go func(reqData *inputJson) {
			downloadVod(dldr, reqData.Url, reqData.OutputFolder, reqData.OutputFile)
		}(reqData)
	}
}

func main() {
	log.SetLevel(log.InfoLevel)
	args, err := parseInputArgs()
	if err != nil {
		log.Errorf("invalid input: %v", err.Error())
	} else {
		if args.url != "" {
			downloadVod(downloader.NewDownloader(), args.url, getDirPath(args.outputFolder), args.outputFile)
		} else {
			downloadVodsFromJson(args.inputFile, args.outputFolder)
		}
	}
}
