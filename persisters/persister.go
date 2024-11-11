package persisters

import "go-gryps/buffers"

type Persister interface {
	Persist(
		userName string,
		mediaData []*buffers.MediaData,
		messagesData []*buffers.MessageData,
	) (string, error)
}
