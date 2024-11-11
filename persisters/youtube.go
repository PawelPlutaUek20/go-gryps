package persisters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"

	"go-gryps/buffers"
)

type YoutubePersister struct {
	service *youtube.Service
}

func NewYoutubePersister() Persister {
	// TODO: get the proper secrets from the new yt account
	googleSecret, err := os.ReadFile("./google-secret-v2.json")
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	youtubeSecret, err := os.ReadFile("./youtube-secret-v2.json")
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	ctx := context.Background()

	config, err := google.ConfigFromJSON(googleSecret, youtube.YoutubeUploadScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	tok := &oauth2.Token{}
	err = json.Unmarshal(youtubeSecret, tok)

	client := config.Client(ctx, tok)

	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Error creating YouTube client: %v", err)
	}

	return &YoutubePersister{
		service: service,
	}
}

func (yp *YoutubePersister) Persist(
	userName string,
	mediaData []*buffers.MediaData,
	messagesData []*buffers.MessageData,
) (string, error) {
	if len(mediaData) == 0 {
		return "", nil
	}

	readers := make([]io.Reader, len(mediaData))
	for i, segment := range mediaData {
		readers[i] = bytes.NewReader(*segment.Data)
	}
	reader := io.MultiReader(readers...)

	var descriptionBuilder strings.Builder
	if len(messagesData) > 0 {
		descriptionBuilder.WriteString(fmt.Sprintf("Grypsy:\n\n"))
	}
	for _, message := range messagesData {
		descriptionBuilder.WriteString(fmt.Sprintf("[%s]: %s\n", message.UserName, message.Message))
	}

	upload := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:       fmt.Sprintf("Nowy grypsiarz: %s", userName),
			Description: descriptionBuilder.String(),
			CategoryId:  "22", // TODO: I have no idea what this is, copied from the docs
		},
		Status: &youtube.VideoStatus{
			PrivacyStatus: "private",
			MadeForKids:   false,
		},
	}

	call := yp.service.Videos.Insert([]string{"snippet,status"}, upload)
	response, err := call.Media(reader).Do()
	if err != nil {
		return "", fmt.Errorf("Failed to uplaod video: %v", err)
	}

	fmt.Printf("Upload successful! videoID: %s\n", response.Id)
	return response.Id, nil
}
