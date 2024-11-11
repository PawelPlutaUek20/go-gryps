package buffers

import (
	"slices"
	"time"
)

type MessageData struct {
	ID       string
	Message  string
	UserName string
	Time     time.Time
}

type MessagesBuffer struct {
	messages    []*MessageData
	maxTimeDiff float64
}

func NewMessagesBuffer(maxTimeDiff int) *MessagesBuffer {
	return &MessagesBuffer{
		messages:    make([]*MessageData, 0, maxTimeDiff),
		maxTimeDiff: float64(maxTimeDiff),
	}
}

func (mb *MessagesBuffer) Insert(message *MessageData) {
	pos := 0
	for i := 0; i < len(mb.messages); i++ {
		msg := mb.messages[i]

		// Dont allow duplicates
		if msg.ID == message.ID {
			return
		}

		if message.Time.Sub(msg.Time).Seconds() <= mb.maxTimeDiff {
			pos = i
			break
		}
	}

	if pos > 0 {
		copy(mb.messages, mb.messages[pos:])
		for i := len(mb.messages) - pos; i < len(mb.messages); i++ {
			mb.messages[i] = nil
		}
		mb.messages = mb.messages[:len(mb.messages)-pos]
	}

	if len(mb.messages) == cap(mb.messages) {
		copy(mb.messages, mb.messages[1:])
		mb.messages[len(mb.messages)-1] = nil
		mb.messages = mb.messages[:len(mb.messages)-1]
	}

	mb.messages = append(mb.messages, message)
}

func (mb *MessagesBuffer) GetByUserName(userName string, limit int) []*MessageData {
	result := make([]*MessageData, 0, 3)

	i := len(mb.messages) - 1

	for i >= 0 && len(result) < 3 {
		msg := mb.messages[i]

		if msg.UserName == userName {
			result = append(result, msg)
		}

		i--
	}

	slices.Reverse(result)
	return result
}
