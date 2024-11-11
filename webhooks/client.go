package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	twitchMessageID        = "twitch-eventsub-message-id"
	twitchMessageTimestamp = "twitch-eventsub-message-timestamp"
	twitchMessageSignature = "twitch-eventsub-message-signature"
	messageType            = "twitch-eventsub-message-type"

	messageTypeVerification = "webhook_callback_verification"
	messageTypeNotification = "notification"
	messageTypeRevocation   = "revocation"

	hmacPrefix = "sha256="
)

type Notification struct {
	Challenge    string `json:"challenge"`
	Subscription struct {
		Type      string          `json:"type"`
		Status    string          `json:"status"`
		Condition json.RawMessage `json:"condition"`
	} `json:"subscription"`
	Event json.RawMessage `json:"event"`
}

type Client struct {
	port   string
	secret string

	onStreamOnline  func()
	onStreamOffline func()
}

func New(port int, secret string) *Client {
	return &Client{
		port:   fmt.Sprintf(":%d", port),
		secret: secret,
	}
}

func (srv *Client) eventSubHandler(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusBadRequest, "Error reading body")
	}

	secret := srv.secret
	message := getHmacMessage(c.Request().Header, body)
	computedHmac := hmacPrefix + getHmac(secret, message)

	if !verifyMessage(computedHmac, c.Request().Header.Get(twitchMessageSignature)) {
		c.Logger().Error("403: Signatures didn't match")
		return c.NoContent(http.StatusForbidden)
	}

	c.Logger().Debug("signatures match")

	var notification Notification
	if err := json.Unmarshal(body, &notification); err != nil {
		c.Logger().Error("Error parsing JSON")
		return c.NoContent(http.StatusBadRequest)
	}

	switch c.Request().Header.Get(messageType) {
	case messageTypeNotification:
		c.Logger().Infof("Event type: %s", notification.Subscription.Type)
		c.Logger().Infof("Event data: %s", string(notification.Event))

		switch notification.Subscription.Type {
		case "stream.online":
			if srv.onStreamOnline != nil {
				srv.onStreamOnline()
			}
		case "stream.offline":
			if srv.onStreamOffline != nil {
				srv.onStreamOffline()
			}
		}

		return c.NoContent(http.StatusNoContent)

	case messageTypeVerification:
		return c.String(http.StatusOK, notification.Challenge)

	case messageTypeRevocation:
		c.Logger().Infof("%s notifications revoked!", notification.Subscription.Type)
		c.Logger().Infof("reason: %s", notification.Subscription.Status)
		c.Logger().Infof("condition: %s", string(notification.Subscription.Condition))
		return c.NoContent(http.StatusNoContent)

	default:
		c.Logger().Infof("Unknown message type: %s", c.Request().Header.Get(messageType))
		return c.NoContent(http.StatusNoContent)
	}
}

func (src *Client) healthcheckHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "available",
	})
}

func (srv *Client) ListenAndServe() {
	e := echo.New()
	e.Pre(middleware.RemoveTrailingSlash())

	e.POST("/eventsub", srv.eventSubHandler)
	e.GET("/healthcheck", srv.healthcheckHandler)

	e.Logger.Infof("Example app listening at http://localhost%s", srv.port)
	e.Logger.Fatal(e.Start(srv.port))
}

func (srv *Client) OnStreamOnline(callback func()) {
	srv.onStreamOnline = callback
}

func (srv *Client) OnStreamOffline(callback func()) {
	srv.onStreamOffline = callback
}

func getHmacMessage(headers http.Header, body []byte) string {
	return headers.Get(twitchMessageID) +
		headers.Get(twitchMessageTimestamp) +
		string(body)
}

func getHmac(secret, message string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

func verifyMessage(hmac1, hmac2 string) bool {
	return hmac.Equal([]byte(hmac1), []byte(hmac2))
}
