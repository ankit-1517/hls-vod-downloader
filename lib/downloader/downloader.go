package downloader

import (
	"bytes"
	"fmt"
	"main/lib/ffmpeg"
	"net/url"
	"os"
	"path"
	"sync"

	"github.com/grafov/m3u8"
	log "github.com/sirupsen/logrus"
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

// returns error occurred, if any
func (dldr *Downloader) DownloadVod(
	manifestUrl string,
	outputPath string,
	outputName string,
) error {
	finalUrl, rsp, err := dldr.fetcher.fetch(manifestUrl)
	if err != nil {
		return err
	}

	playlist, playlistType, err := m3u8.DecodeFrom(bytes.NewReader(rsp), false)
	if err != nil {
		log.Errorf("error decoding manifest from rsp %v: %v", manifestUrl, err.Error())
		return err
	}

	if playlistType == m3u8.MASTER {
		masterPlaylist := playlist.(*m3u8.MasterPlaylist)
		log.Debugf("successfully downloaded master manifest for %v url %v: %v", outputName, manifestUrl, masterPlaylist.String())
		return dldr.downloadFromMasterManifest(masterPlaylist, finalUrl, outputPath, outputName)
	} else if playlistType == m3u8.MEDIA {
		indexPlaylist := playlist.(*m3u8.MediaPlaylist)
		log.Debugf("successfully downloaded index manifest for %v url %v: %v", outputName, manifestUrl, indexPlaylist.String())
		return dldr.downloadFromIndexManifest(indexPlaylist, finalUrl, outputPath, outputName)
	}
	return fmt.Errorf("expected playlistType %v or %v, recvd %v", m3u8.MASTER, m3u8.MEDIA, playlistType)
}

func (dldr *Downloader) downloadFromMasterManifest(
	masterManifest *m3u8.MasterPlaylist,
	finalMasterUrl string,
	outputPath string,
	outputName string,
) error {
	// TODO: Add a better way to choose variant to be downloaded
	variant := findMaxBandwidthVariant(masterManifest.Variants)
	if variant == nil {
		err := fmt.Errorf("unable to determine variant to be fetched!")
		log.Error(err.Error())
		return err
	}
	variantUrl := resolveReference(finalMasterUrl, variant.URI)
	return dldr.downloadFromIndexUrl(variantUrl, outputPath, outputName)
}

func (dldr *Downloader) downloadFromIndexUrl(
	indexUrl string,
	outputPath string,
	outputName string,
) error {
	finalUrl, indexManifest, err := dldr.fetcher.fetchIndexManifest(indexUrl)
	if err != nil {
		return err
	}

	log.Debugf("successfully downloaded index manifest for %v url %v: %v", outputName, indexUrl, indexManifest.String())

	return dldr.downloadFromIndexManifest(indexManifest, finalUrl, outputPath, outputName)
}

func (dldr *Downloader) downloadFromIndexManifest(
	variant *m3u8.MediaPlaylist,
	url string,
	outputPath string,
	outputName string,
) error {
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
	baseUrl string,
	outputPath string,
	outputName string,
) ([]string, error) {
	outputPaths := make([]string, 0)

	sem := make(chan struct{}, dldr.concurrentDownloads)
	var wg sync.WaitGroup
	failed := false

	for idx, segment := range variant.Segments {
		if failed || segment == nil {
			continue
		}
		idxPath := path.Join(outputPath, fmt.Sprintf("%v_%v.ts", outputName, idx))
		outputPaths = append(outputPaths, idxPath)
		wg.Add(1)
		sem <- struct{}{}
		go func(segment *m3u8.MediaSegment, path string) {
			err := dldr.downloadSegment(segment, baseUrl, path)
			if err != nil {
				failed = true
			}
			<-sem
			wg.Done()
		}(segment, idxPath)
	}
	wg.Wait()

	if failed {
		err := fmt.Errorf("some error occurred when fetching segments")
		log.Error(err.Error())
		return nil, err
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
		log.Errorf("error creating file %v: %v", outputPath, err.Error())
		return err
	}
	defer file.Close()
	_, err = file.Write(rsp)
	if err != nil {
		log.Errorf("error writing to file %v: %v", outputPath, err.Error())
		return err
	}
	return nil
}
