package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
)

type args struct {
	url          string
	outputFolder string
	outputFile   string
	inputFile    string
}

func exists(key string, mp map[string]struct{}) bool {
	_, ok := mp[key]
	return ok
}

func parseInputArgs() (*args, error) {
	urlFlag := "u"
	inputFileFlag := "j"
	outputFolderFlag := "o"
	outputFileFlag := "n"

	url := flag.String(urlFlag, "", "url of m3u8 to be downloaded")
	inputFile := flag.String(inputFileFlag, "", "path to input json")
	outputFolder := flag.String(outputFolderFlag, "output", "output folder")
	outputFile := flag.String(outputFileFlag, "video.mp4", "name to save downloaded video file as")

	flag.Parse()
	flagsPassed := make(map[string]struct{}, 0)
	flag.Visit(func(f *flag.Flag) {
		flagsPassed[f.Name] = struct{}{}
	})

	if exists(urlFlag, flagsPassed) && exists(inputFileFlag, flagsPassed) {
		return nil, fmt.Errorf("invalid args: both urlFlag and inputFileFlag present")
	} else if !exists(urlFlag, flagsPassed) && !exists(inputFileFlag, flagsPassed) {
		return nil, fmt.Errorf("invalid args: neither of urlFlag and inputFileFlag present")
	} else if exists(inputFileFlag, flagsPassed) && exists(outputFileFlag, flagsPassed) {
		log.Warnln("ignoring outputFileFlag since using inputFileFlag")
	}

	inputArgs := &args{
		*url,
		*outputFolder,
		*outputFile,
		*inputFile,
	}
	return inputArgs, nil
}

type inputJson struct {
	Url          string `json:"url"`
	OutputFolder string `json:"outputFolder"`
	OutputFile   string `json:"outputFile"`
}

func fillEmptyData(data []*inputJson) {
	for idx, d := range data {
		if d.OutputFile == "" {
			d.OutputFile = fmt.Sprintf("video_%v.mp4", idx+1)
		}
		if d.OutputFolder == "" {
			d.OutputFolder = "output"
		}
	}
}

func getDirPath(outputFolder string) string {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return path.Join(pwd, outputFolder)
}

func dedupOutputFileNames(data []*inputJson) {
	for _, d := range data {
		d.OutputFolder = getDirPath(d.OutputFolder)
	}
	for idx, d := range data {
		time.Sleep(time.Millisecond)
		for idxPrev := 0; idxPrev < idx; idxPrev += 1 {
			if d.OutputFolder == data[idxPrev].OutputFolder && d.OutputFile == data[idxPrev].OutputFile {
				d.OutputFile = fmt.Sprintf("%v_%v", time.Now().Nanosecond(), d.OutputFile)
				log.Infof("renaming %v/%v to %v\n", d.OutputFolder, data[idxPrev].OutputFile, d.OutputFile)
				break
			}
		}
	}
}

func parseInputJson(path string, outputFolder string) ([]*inputJson, error) {
	var data []*inputJson
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	err = json.Unmarshal(file, &data)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	fillEmptyData(data)
	dedupOutputFileNames(data)
	return data, nil
}
