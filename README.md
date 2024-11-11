# Go gryps

## Refresh token

`https://github.com/youtube/api-samples/blob/master/go/oauth2.go`

To get a google oauth2 token with a refersh_token, do this:

```
go run cmd/oauth/main.go
```

To trigger a `stream.online` webhook, do this:

```
twitch event trigger streamup -F http://localhost:8080/eventsub/ -s "your secret goes here"
```

To trigger a `stream.offline` webhook, do this:

```
twitch event trigger streamdown -F http://localhost:8080/eventsub/ -s "your secret goes here"
```

To trigger ngrok, do this:

```
ngrok config add-authtoken $YOUR_AUTHTOKEN
ngrok http --url=giraffe-whole-dragon.ngrok-free.app 8080
```
