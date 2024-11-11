package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/gempir/go-twitch-irc/v4"
	"github.com/go-resty/resty/v2"

	"go-gryps/buffers"
	"go-gryps/hls"
	"go-gryps/persisters"
	"go-gryps/utils"
	"go-gryps/webhooks"
)

type config struct {
	env    string
	secret string
	port   int
	twitch struct {
		channel string
	}
}

type application struct {
	config config

	logger *slog.Logger

	twitchClient  *twitch.Client
	restyClient   *resty.Client
	hlsClient     *hls.Client
	webhookClient *webhooks.Client

	persister persisters.Persister

	mediaBuffer    *buffers.MediaBuffer
	messagesBuffer *buffers.MessagesBuffer
}

func main() {
	var cfg config

	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.twitch.channel, "twitch-channel", "xqc", "Twitch channel")
	flag.IntVar(&cfg.port, "port", 8080, "Webhook client port")
	flag.StringVar(&cfg.secret, "secret", "your secret goes here", "Webhook client secret")

	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelDebug)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	}))

	restyClient := resty.New()
	twitchClient := twitch.NewAnonymousClient()
	persister := persisters.NewYoutubePersister()
	hlsClient := hls.NewTwitchHLSClient()
	webhookClient := webhooks.New(cfg.port, cfg.secret)

	app := &application{
		config:        cfg,
		logger:        logger,
		twitchClient:  twitchClient,
		restyClient:   restyClient,
		persister:     persister,
		hlsClient:     hlsClient,
		webhookClient: webhookClient,
	}

	app.mediaBuffer = buffers.NewMediaBuffer(90)
	app.messagesBuffer = buffers.NewMessagesBuffer(600)

	app.start()
}

func (app *application) start() {
	var cancelFunc context.CancelFunc

	go app.listenToMessages()

	app.webhookClient.OnStreamOnline(func() {
		app.logger.Info("Stream went online")

		if cancelFunc != nil {
			cancelFunc()
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancelFunc = cancel

		go app.listenToStream(ctx)
	})

	app.webhookClient.OnStreamOffline(func() {
		app.logger.Info("Stream went offline")
		app.twitchClient.Disconnect()

		if cancelFunc != nil {
			cancelFunc()
		}
	})

	app.webhookClient.ListenAndServe()
}

func (app *application) listenToStream(ctx context.Context) error {
	app.mediaBuffer = buffers.NewMediaBuffer(90)

	err := app.hlsClient.Join(app.config.twitch.channel)
	if err != nil {
		app.logger.Error("Failed to join stream", "err", err)
		return err
	}

	app.hlsClient.OnMediaSegmentWithBytes(func(media hls.MediaSegmentWithBytes) {
		app.logger.Debug("New media segment fetched", "SeqId", media.MediaSegment.SeqId)
		mediaData := &buffers.MediaData{
			SeqId:    media.MediaSegment.SeqId,
			Data:     media.Bytes,
			Duration: media.MediaSegment.Duration,
		}
		app.mediaBuffer.Insert(mediaData)
	})

	// TODO: instead of passing a context I should probably add a Close method
	err = app.hlsClient.Connect(ctx)
	if err != nil {
		app.logger.Error("Failed during hls", "err", err)
		return err
	}

	return nil
}

func (app *application) persistStream(userName string) error {
	app.logger.Info("Persisting stream...")
	messages := app.messagesBuffer.GetByUserName(userName, 3)
	media := app.mediaBuffer.Segments()

	_, err := app.persister.Persist(userName, media, messages)
	if err != nil {
		app.logger.Error("Failed to persist stream", "err", err)
	}

	return err
}

func (app *application) listenToMessages() error {
	app.messagesBuffer = buffers.NewMessagesBuffer(600)

	throttledPersist := utils.Throttle(utils.Delay(app.persistStream, 30*time.Second), 60*time.Second)
	app.twitchClient.Join(app.config.twitch.channel)

	app.twitchClient.OnConnect(func() {
		app.logger.Info("Connected to server")
	})

	app.twitchClient.OnClearChatMessage(func(message twitch.ClearChatMessage) {
		app.logger.Debug("clear chat message", "message", message.Message)
		go throttledPersist(message.TargetUsername)
	})

	app.twitchClient.OnClearMessage(func(message twitch.ClearMessage) {
		app.logger.Debug("clear message", "message", message.Message)
		go throttledPersist(message.Login)
	})

	app.twitchClient.OnPrivateMessage(func(message twitch.PrivateMessage) {
		app.logger.Debug("private message", "message", message.Message)
		messageData := &buffers.MessageData{
			ID:       message.ID,
			Time:     message.Time,
			Message:  message.Message,
			UserName: message.User.Name,
		}
		app.messagesBuffer.Insert(messageData)
	})

	return app.twitchClient.Connect()
}
