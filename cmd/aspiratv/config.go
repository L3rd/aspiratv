package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/simulot/aspiratv/providers"
)

func (a *app) Initialize() {
	err := a.ReadConfig(a.Config.ConfigFile)
	if err != nil {
		a.logger.Fatal().Printf("[Initialize] %s", err)
	}

	// Check ans normalize configuration file
	a.Config.Check()

	// Check ffmpeg presence
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("where", "ffmpeg")
	} else {
		cmd = exec.Command("which", "ffmpeg")
	}
	b, err := cmd.Output()
	if err != nil {
		a.logger.Fatal().Printf("[Initialize] Can't determine ffmpeg path: %s", err)
	}
	a.ffmpeg = strings.Trim(strings.Trim(string(b), "\r\n"), "\n")
	a.logger.Trace().Printf("[Initialize] FFMPEG path: %q", a.ffmpeg)

	// Get FFMPEG version
	cmd = exec.Command(a.ffmpeg, "-version")
	b, err = cmd.Output()
	a.logger.Debug().Printf("[Initialize] FFMPEG version: %q", string(b))
}

type ProviderConfig struct {
	Enabled  bool
	Settings map[string]string
}

// Handle Duration as string for JSON configuration
type textDuration time.Duration

func (t textDuration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Duration(t).String() + `"`), nil
}

func (t *textDuration) UnmarshalJSON(b []byte) error {
	if b[0] == '"' {
		b = b[1 : len(b)-1]
	}
	v, err := time.ParseDuration(string(b))
	if err != nil {
		return err
	}
	*t = textDuration(v)
	return nil
}

// Almost empty configuration for testing purpose
var defaultConfig = &config{
	WatchList: []*providers.MatchRequest{
		{
			Provider:    "francetv",
			Show:        "Les Lapins Crétins",
			Destination: "Jeunesse",
		},
	},
	Destinations: map[string]string{
		"Jeunesse": "${HOME}/Videos/Jeunesse",
	},
}

// WriteConfig create a JSON file with the current configuration
func WriteConfig() {
	f, err := os.Create("config.json")
	if err != nil {
		log.Fatalf("Can't write configuration file: %v", err)
	}
	e := json.NewEncoder(f)
	e.SetIndent("", "  ")
	e.Encode(defaultConfig)
	f.Close()
	os.Exit(0)
}

// ReadConfig read the JSON configuration file
func (a *app) ReadConfig(configFile string) error {
	a.logger.Trace().Printf("[ReadConfig] opening '%s'", configFile)
	f, err := os.Open(configFile)
	if err != nil {
		return fmt.Errorf("Can't open configuration file: %v", err)
	}
	defer f.Close()
	d := json.NewDecoder(f)
	err = d.Decode(&a.Config)
	if err != nil {
		return fmt.Errorf("Can't decode configuration file: %v", err)
	}
	return nil
}

// Check the configuration or die
func (c *config) Check() {

	// Expand paths
	for d, p := range c.Destinations {
		c.Destinations[d] = os.ExpandEnv(p)
	}

	for _, m := range c.WatchList {
		m.Pitch = strings.ToLower(m.Pitch)
		m.Show = strings.ToLower(m.Show)
		m.Title = strings.ToLower(m.Title)
		if _, ok := c.Destinations[m.Destination]; !ok {
			log.Fatalf("Destination %q for show %q is not defined into section Destination of %q", m.Destination, m.Show, c.ConfigFile)
		}
	}

}

func (c *config) IsProviderActive(p string) bool {
	if pc, ok := c.Providers[p]; ok {
		return pc.Enabled
	}
	return false
}
