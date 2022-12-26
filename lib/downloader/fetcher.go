package downloader

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/grafov/m3u8"
	"github.com/hashicorp/go-retryablehttp"

	log "github.com/sirupsen/logrus"
)

type fetcher struct {
	client *retryablehttp.Client
}

func newFetcher() *fetcher {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Logger = nil
	return &fetcher{
		client: retryClient,
	}
}

func (ft *fetcher) fetch(url string) (string, []byte, error) {
	log.Debugf("get req for url: %v", url)
	rsp, err := ft.client.Get(url)
	if err != nil {
		return "", nil, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("rcvd status code %v, expected %v", rsp.StatusCode, http.StatusOK)
	}

	rspBytes, err := io.ReadAll(rsp.Body)
	if err != nil {
		return "", nil, err
	}
	return rsp.Request.URL.String(), rspBytes, nil
}

/*
Returns:
1. Final url (possibly redirected) -- useful for fetching index files
2. Master manifest file
3. Error occured
*/
func (ft *fetcher) fetchMasterManifest(url string) (string, *m3u8.MasterPlaylist, error) {
	finalUrl, rsp, err := ft.fetch(url)
	if err != nil {
		log.Errorf("error fetching manifest %v: %v", url, err.Error())
		return "", nil, err
	}

	playlist, playlistType, err := m3u8.DecodeFrom(bytes.NewReader(rsp), false)
	if err != nil {
		log.Errorf("error decoding manifest from rsp %v: %v", url, err.Error())
		return "", nil, err
	}
	if playlistType != m3u8.MASTER {
		err = fmt.Errorf("rcvd file type %v for vod %v, expected %v", playlistType, url, m3u8.MASTER)
		log.Errorf(err.Error())
		return "", nil, err
	}

	masterPlaylist := playlist.(*m3u8.MasterPlaylist)
	return finalUrl, masterPlaylist, nil
}

/*
Returns:
1. Final url (possibly redirected) -- useful for fetching segment files
2. Media index file
3. Error occured
*/
func (ft *fetcher) fetchIndexManifest(url string) (string, *m3u8.MediaPlaylist, error) {
	finalUrl, rsp, err := ft.fetch(url)
	if err != nil {
		log.Errorf("error fetching manifest %v: %v", url, err.Error())
		return "", nil, err
	}

	playlist, playlistType, err := m3u8.DecodeFrom(bytes.NewReader(rsp), false)
	if err != nil {
		log.Errorf("error decoding manifest from rsp %v: %v", url, err.Error())
		return "", nil, err
	}
	if playlistType != m3u8.MEDIA {
		err = fmt.Errorf("rcvd file type %v for vod %v, expected %v", playlistType, url, m3u8.MEDIA)
		log.Errorf(err.Error())
		return "", nil, err
	}

	mediaPlaylist := playlist.(*m3u8.MediaPlaylist)
	return finalUrl, mediaPlaylist, nil
}

/*
Returns:
1. Response body in a byte array
2. Error occured
*/
func (ft *fetcher) fetchSegment(url string) ([]byte, error) {
	_, rsp, err := ft.fetch(url)
	if err != nil {
		log.Errorf("error fetching segment %v: %v", url, err.Error())
		return nil, err
	}
	return rsp, nil
}
