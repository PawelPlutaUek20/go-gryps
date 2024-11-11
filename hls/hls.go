package hls

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/grafov/m3u8"
)

type hlsClient struct {
	MasterPlaylistURI string
	lastSegments      []*m3u8.MediaSegment
	restyClient       *resty.Client

	onMediaSegmentWithBytes func(mediaSegmentWithBytes MediaSegmentWithBytes)
}

type MediaSegmentWithBytes struct {
	MediaSegment *m3u8.MediaSegment
	Bytes        *[]byte
}

func newHlsClient() *hlsClient {
	restyClient := resty.New()

	return &hlsClient{
		lastSegments: make([]*m3u8.MediaSegment, 0),
		restyClient:  restyClient,
	}
}

func (hls *hlsClient) Run(ctx context.Context) error {
	masterPlaylist, err := hls.getMasterPlaylist(hls.MasterPlaylistURI)
	if err != nil {
		return err
	}

	//TODO: I hope Variants[0] is the best quality
	masterPlaylistURI := masterPlaylist.Variants[0].URI

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			mediaPlaylist, err := hls.getMediaPlaylist(masterPlaylistURI)
			if err != nil {
				return err
			}

			if mediaPlaylist.Closed {
				return nil
			}

			hls.getPlaylistSegments(mediaPlaylist.Segments)

			// TODO: this is not accurate
			time.Sleep(time.Duration(mediaPlaylist.TargetDuration) * time.Second)
		}
	}
}

func (hls *hlsClient) getMasterPlaylist(URI string) (*m3u8.MasterPlaylist, error) {
	resp, err := hls.restyClient.R().SetDoNotParseResponse(true).Get(URI)
	if err != nil {
		return nil, err
	}

	rawBody := resp.RawBody()
	defer rawBody.Close()

	playlist, _, err := m3u8.DecodeFrom(rawBody, true)
	if err != nil {
		return nil, err
	}

	masterPlaylist := playlist.(*m3u8.MasterPlaylist)
	return masterPlaylist, nil
}

func (hls *hlsClient) getMediaPlaylist(masterPlaylistURI string) (*m3u8.MediaPlaylist, error) {
	resp, err := hls.restyClient.R().SetDoNotParseResponse(true).Get(masterPlaylistURI)
	if err != nil {
		return nil, err
	}

	rawBody := resp.RawBody()
	defer rawBody.Close()

	playlist, _, err := m3u8.DecodeFrom(rawBody, true)
	if err != nil {
		return nil, err
	}

	mediaPlaylist := playlist.(*m3u8.MediaPlaylist)
	return mediaPlaylist, nil
}

func (hls *hlsClient) getPlaylistSegments(playlistSegments []*m3u8.MediaSegment) {
	var wg sync.WaitGroup

	for i, playlistSegment := range playlistSegments {
		if playlistSegment == nil {
			break
		}

		if slices.ContainsFunc(hls.lastSegments, func(segment *m3u8.MediaSegment) bool {
			return segment != nil && segment.SeqId == playlistSegment.SeqId
		}) {
			continue
		}

		wg.Add(1)
		go func(i int, playlistSegment *m3u8.MediaSegment) {
			defer wg.Done()

			data, err := hls.getMediaSegmentURI(playlistSegment.URI)
			if err != nil {
				return
			}

			mediaData := MediaSegmentWithBytes{
				MediaSegment: playlistSegment,
				Bytes:        &data,
			}

			if hls.onMediaSegmentWithBytes != nil {
				hls.onMediaSegmentWithBytes(mediaData)
			}

		}(i, playlistSegment)
	}

	wg.Wait()
	hls.lastSegments = playlistSegments
}

func (hls *hlsClient) getMediaSegmentURI(segmentURI string) ([]byte, error) {
	response, err := hls.restyClient.R().Get(segmentURI)
	if err != nil {
		return nil, err
	}

	return response.Body(), err
}
