package version

const (
	AppName     = "Spotify AdFree"
	GitHubOwner = "Leskur"
	GitHubRepo  = "spotify-adfree"
)

// Version 本地开发默认值；发版可由 -ldflags -X 注入。
var Version = "0.0.1"
