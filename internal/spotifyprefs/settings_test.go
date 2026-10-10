package spotifyprefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := Paths{Backup: filepath.Join(dir, "spotify-prefs.bak")}

	if err := p.SaveManaged(true); err != nil {
		t.Fatal(err)
	}
	if err := p.SaveStartSpotifyOnLaunch(true); err != nil {
		t.Fatal(err)
	}
	if !p.LoadManaged() || !p.LoadStartSpotifyOnLaunch() || p.LoadQuitSpotifyOnExit() {
		t.Fatalf("unexpected settings: %+v", p.LoadSettings())
	}
	if _, err := os.Stat(p.settingsPath()); err != nil {
		t.Fatal("settings.json missing")
	}
}
