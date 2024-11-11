package persisters

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"go-gryps/buffers"
)

type LocalPersister struct{}

func NewLocalPersister() Persister {
	return &LocalPersister{}
}

func (p *LocalPersister) Persist(
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

	timestamp := time.Now().Format("2006-01-02_150405")
	path := fmt.Sprintf("%s.ts", timestamp)

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}

	defer f.Close()

	_, err = io.Copy(f, reader)
	if err != nil {
		return "", err
	}

	return "", nil
}
