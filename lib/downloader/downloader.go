package downloader

import (
	"fmt"
	"main/lib/ffmpeg"
	"net/url"
	"os"
	"path"
	"sync"

	"github.com/grafov/m3u8"
)

type Downloader struct {
	*fetcher
	fmHelper            *ffmpeg.FMHelper
	concurrentDownloads int // num of segments downloaded concurrently
}

func NewDownloader() *Downloader {
	return &Downloader{
		fetcher:             newFetcher(),
		fmHelper:            ffmpeg.NewFMHelper(),
		concurrentDownloads: 50,
	}
}

func findMaxBandwidthVariant(variants []*m3u8.Variant) *m3u8.Variant {
	maxBw := uint32(0)
	var variantMax *m3u8.Variant
	for _, variant := range variants {
		if variant == nil {
			continue
		}
		if variant.Bandwidth > maxBw {
			maxBw = variant.Bandwidth
			variantMax = variant
		}
	}
	return variantMax
}

func resolveReference(base, ref string) string {
	baseUrl, _ := url.Parse(base)
	refUrl, _ := url.Parse(ref)
	return baseUrl.ResolveReference(refUrl).String()
}

func getDirPath(outputFolder string) string {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return path.Join(pwd, outputFolder)
}

// returns error occurred, if any
func (dldr *Downloader) DownloadFromMasterUrl(
	masterUrl string,
	outputPath string,
	outputName string,
) error {
	finalUrl, masterManifest, err := dldr.fetcher.fetchMasterManifest(masterUrl)
	if err != nil {
		return err
	}

	// TODO: Add a better way to choose variant to be downloaded
	variant := findMaxBandwidthVariant(masterManifest.Variants)
	if variant == nil {
		return fmt.Errorf("unable to determine variant to be fetched!")
	}

	variantUrl := resolveReference(finalUrl, variant.URI)
	return dldr.downloadFromIndexUrl(variantUrl, outputPath, outputName)
}

func (dldr *Downloader) downloadFromIndexUrl(
	url string,
	outputPath string,
	outputName string,
) error {
	finalUrl, indexManifest, err := dldr.fetcher.fetchIndexManifest(url)
	if err != nil {
		return err
	}
	return dldr.downloadFromIndexManifest(indexManifest, finalUrl, outputPath, outputName)
}

func (dldr *Downloader) downloadFromIndexManifest(
	variant *m3u8.MediaPlaylist,
	url string,
	outputPath string,
	outputName string,
) error {
	outputPath = getDirPath(outputPath)

	segments, err := dldr.downloadSegments(variant, url, outputPath, outputName)
	if err != nil {
		return err
	}

	return dldr.joinSegments(segments, path.Join(outputPath, outputName), outputPath)
}

func (dldr *Downloader) joinSegments(
	inputFiles []string,
	outputFile string,
	outputPath string,
) error {
	return dldr.fmHelper.ConvertSegmentsToMp4(inputFiles, outputFile, outputPath)
}

func (dldr *Downloader) downloadSegments(
	variant *m3u8.MediaPlaylist,
	url string,
	outputPath string,
	outputName string,
) ([]string, error) {
	outputPaths := make([]string, 0)

	sem := make(chan struct{}, dldr.concurrentDownloads)
	var wg sync.WaitGroup
	failed := false

	for idx, segment := range variant.Segments {
		if segment == nil {
			continue
		}
		idxPath := path.Join(outputPath, fmt.Sprintf("%v_%v.ts", outputName, idx))
		outputPaths = append(outputPaths, idxPath)
		wg.Add(1)
		sem <- struct{}{}
		go func(segment *m3u8.MediaSegment, path string) {
			err := dldr.downloadSegment(segment, url, path)
			if err != nil {
				fmt.Printf("error occured downloading segment: %v", err.Error())
				failed = true
			}
			<-sem
			wg.Done()
		}(segment, idxPath)
	}
	wg.Wait()

	if failed {
		return nil, fmt.Errorf("error occured fetching segments")
	}
	return outputPaths, nil
}

func (dldr *Downloader) downloadSegment(segment *m3u8.MediaSegment, url string, outputPath string) error {
	rsp, err := dldr.fetchSegment(resolveReference(url, segment.URI))
	if err != nil {
		return err
	}
	return dldr.saveToDisk(rsp, outputPath)
}

func (dldr *Downloader) saveToDisk(rsp []byte, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(rsp)
	return err
}
