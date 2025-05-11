package webhooks

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestWebhooks(t *testing.T) {

	tests := []struct {
		secret              string
		messageID           string
		messageTimestamp    string
		signature           string
		messageType         string
		body                string
		expectedCode        int
		expectedBody        string
		expectedContentType string
	}{
		{
			secret:           "your secret goes here",
			messageID:        "cb1b5f98-00cb-4183-2d85-31ed5ff80b6e",
			messageTimestamp: "2025-05-11T13:40:02.2895535Z",
			signature:        "sha256=f7879de88ad65b9c02143cba87da22dc0a54a0edbf8989c504da9db157073b8b",
			body:             `{"subscription":{"id":"cb1b5f98-00cb-4183-2d85-31ed5ff80b6e","status":"enabled","type":"stream.online","version":"1","condition":{"broadcaster_user_id":"18940026"},"transport":{"method":"webhook","callback":"null"},"created_at":"2025-05-11T13:40:02.2895535Z","cost":0},"event":{"id":"17759201","broadcaster_user_id":"18940026","broadcaster_user_login":"testBroadcaster","broadcaster_user_name":"testBroadcaster","type":"live","started_at":"2025-05-11T13:40:02.2895535Z"}}`,
			messageType:      messageTypeNotification,
			expectedCode:     http.StatusNoContent,
		},
		{
			secret:              "your secret goes here",
			messageID:           "d3313a8f-052e-7ecd-4808-b91987ff1a3d",
			messageTimestamp:    "2025-05-11T16:37:20.8486018Z",
			signature:           "sha256=56e1799068b584fe519993dd163203f681c178e937d173e63dee2b499ef33896",
			body:                `{"challenge":"a4a07492-1256-c6e9-d96e-964ff8327e59","subscription":{"id":"d3313a8f-052e-7ecd-4808-b91987ff1a3d","status":"webhook_callback_verification_pending","type":"stream.online","version":"1","condition":{"broadcaster_user_id":"65542233"},"transport":{"method":"webhook","callback":"http://localhost:8080/eventsub/"},"created_at":"2025-05-11T16:37:20.8486018Z","cost":0}}`,
			messageType:         messageTypeVerification,
			expectedBody:        "a4a07492-1256-c6e9-d96e-964ff8327e59",
			expectedContentType: "text/plain; charset=UTF-8",
			expectedCode:        http.StatusOK,
		},
	}

	for _, tt := range tests {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/eventsub", strings.NewReader(tt.body))
		req.Header.Set(twitchMessageID, tt.messageID)
		req.Header.Set(twitchMessageTimestamp, tt.messageTimestamp)
		req.Header.Set(twitchMessageSignature, tt.signature)
		req.Header.Set(messageType, tt.messageType)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := New(0, tt.secret)

		if assert.NoError(t, h.eventSubHandler(c)) {
			assert.Equal(t, tt.expectedCode, rec.Code)
			assert.Equal(t, tt.expectedBody, rec.Body.String())
			assert.Equal(t, tt.expectedContentType, rec.Header().Get(echo.HeaderContentType))
		}
	}

}
