package spotifyprefs

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	BlockAds             bool `json:"block_ads"`
	StartSpotifyOnLaunch bool `json:"start_spotify_on_launch"`
	QuitSpotifyOnExit    bool `json:"quit_spotify_on_exit"`
}

func (p Paths) settingsPath() string {
	return filepath.Join(filepath.Dir(p.Backup), "settings.json")
}

func (p Paths) LoadSettings() Settings {
	data, err := os.ReadFile(p.settingsPath())
	if err != nil {
		return Settings{}
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}
	}
	return s
}

func (p Paths) SaveSettings(s Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(p.settingsPath(), data, 0o644)
}

func (p Paths) LoadManaged() bool {
	return p.LoadSettings().BlockAds
}

func (p Paths) SaveManaged(on bool) error {
	s := p.LoadSettings()
	s.BlockAds = on
	return p.SaveSettings(s)
}

func (p Paths) LoadQuitSpotifyOnExit() bool {
	return p.LoadSettings().QuitSpotifyOnExit
}

func (p Paths) SaveQuitSpotifyOnExit(on bool) error {
	s := p.LoadSettings()
	s.QuitSpotifyOnExit = on
	return p.SaveSettings(s)
}

func (p Paths) LoadStartSpotifyOnLaunch() bool {
	return p.LoadSettings().StartSpotifyOnLaunch
}

func (p Paths) SaveStartSpotifyOnLaunch(on bool) error {
	s := p.LoadSettings()
	s.StartSpotifyOnLaunch = on
	return p.SaveSettings(s)
}
