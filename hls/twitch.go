package hls

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	restyClient *resty.Client
	hlsClient   *hlsClient

	channel     string
	accessToken *streamPlaybackAccessToken
}

func NewTwitchHLSClient() *Client {
	restyClient := resty.New()
	hlsClient := newHlsClient()

	return &Client{
		restyClient: restyClient,
		hlsClient:   hlsClient,
	}
}

func (c *Client) Join(channel string) error {
	c.channel = channel

	accessToken, err := c.getAccessToken(c.channel)
	if err != nil {
		return err
	}

	c.accessToken = accessToken
	return nil
}

func (c *Client) OnMediaSegmentWithBytes(callback func(mediaSegmentWithBytes MediaSegmentWithBytes)) {
	c.hlsClient.onMediaSegmentWithBytes = callback
}

func (c *Client) Connect(ctx context.Context) error {
	c.hlsClient.MasterPlaylistURI = c.fmtMasterPlaylistURI()
	return c.hlsClient.Run(ctx)
}

func (c *Client) fmtMasterPlaylistURI() string {
	endpoint := fmt.Sprintf("/api/channel/hls/%s.m3u8", c.channel)

	params := url.Values{
		"p":                []string{strconv.Itoa(rand.Intn(1000000))},
		"type":             []string{"any"},
		"allow_source":     []string{"true"},
		"allow_audio_only": []string{"true"},
		"allow_spectre":    []string{"false"},
		"sig":              []string{c.accessToken.Signature},
		"token":            []string{c.accessToken.Value},
		"fast_bread":       []string{"true"},
	}

	url := url.URL{
		Scheme:   "https",
		Host:     "usher.ttvnw.net",
		Path:     endpoint,
		RawQuery: params.Encode(),
	}

	return url.String()
}

type graphQLExtensions struct {
	PersistedQuery graphQLPersistedQuery `json:"persistedQuery"`
}

type graphQLPersistedQuery struct {
	Version    int    `json:"version"`
	Sha256Hash string `json:"sha256Hash"`
}

type graphQLQuery struct {
	OperationName string            `json:"operationName"`
	Extensions    graphQLExtensions `json:"extensions"`
	Variables     interface{}       `json:"variables"`
}

type playbackAcessTokenVariables struct {
	Login      string `json:"login"`
	PlayerType string `json:"playerType"`
	VodID      string `json:"vodID"`
	IsLive     bool   `json:"isLive"`
	IsVod      bool   `json:"isVod"`
}

type playbackAccessTokenGraphQLResponse struct {
	Data playbackAccessTokenGraphQLData `json:"data"`
}

type playbackAccessTokenGraphQLData struct {
	StreamPlaybackAccessToken streamPlaybackAccessToken `json:"streamPlaybackAccessToken"`
}

type streamPlaybackAccessToken struct {
	Signature string `json:"signature"`
	Value     string `json:"value"`
}

func (c *Client) getAccessToken(channel string) (*streamPlaybackAccessToken, error) {
	gqlURL := "https://gql.twitch.tv/gql"

	query := graphQLQuery{
		OperationName: "PlaybackAccessToken",
		Extensions: graphQLExtensions{
			PersistedQuery: graphQLPersistedQuery{
				Version:    1,
				Sha256Hash: "0828119ded1c13477966434e15800ff57ddacf13ba1911c129dc2200705b0712",
			},
		},
		Variables: playbackAcessTokenVariables{
			IsLive:     true,
			Login:      channel,
			IsVod:      false,
			VodID:      "",
			PlayerType: "embed",
		},
	}

	var result playbackAccessTokenGraphQLResponse

	_, err := c.restyClient.
		R().
		SetHeader("Client-ID", "kimne78kx3ncx6brgo4mv6wki5h1ko").
		SetHeader("Content-Type", "application/json").
		SetBody(query).
		SetResult(&result).
		Post(gqlURL)

	if err != nil {
		return nil, err
	}

	return &result.Data.StreamPlaybackAccessToken, nil
}
