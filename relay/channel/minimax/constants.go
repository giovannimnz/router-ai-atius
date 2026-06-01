package minimax

// https://www.minimaxi.com/document/guides/chat-model/V2?id=65e0736ab2845de20908e2dd
// Only canonical model names appear in /v1/models. The -hs variants are
// accepted at the routing layer (see model_mapping) and rewritten to
// -highspeed upstream, but they are not exposed as separate catalog entries.

var ModelList = []string{
	"abab6.5-chat",
	"abab6.5s-chat",
	"abab6-chat",
	"abab5.5-chat",
	"abab5.5s-chat",
	"MiniMax-M3",
	"MiniMax-M2.7",
	"MiniMax-M2.7-highspeed",
	"speech-2.5-hd-preview",
	"speech-2.5-turbo-preview",
	"speech-02-hd",
	"speech-02-turbo",
	"speech-01-hd",
	"speech-01-turbo",
	"MiniMax-M2.1",
	"MiniMax-M2.1-highspeed",
	"MiniMax-M2",
	"MiniMax-M2.5",
	"MiniMax-M2.5-highspeed",
	"image-01",
	"image-01-live",
}

var ChannelName = "minimax"
