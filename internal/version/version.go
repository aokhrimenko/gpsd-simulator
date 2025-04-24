package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aokhrimenko/gpsd-simulator/internal/logger"
)

const (
	latestVersionApiEndpoint = "https://api.github.com/repos/aokhrimenko/gpsd-simulator/releases/latest"
	retries                  = 3
	retryTimeout             = 5 * time.Second
)

type githubLatestRelease struct {
	Id        int       `json:"id"`
	HtmlUrl   string    `json:"html_url"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Body      string    `json:"body"`
}

func CheckForUpdate(ctx context.Context, log logger.Logger, currentVersion string) {
	client := &http.Client{Transport: &http.Transport{}}

	for i := 0; i < retries; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		latestRelease, err := fetch(client)
		if err != nil {
			log.Debugf("Version: error fetching latest release: %v", err)
			time.Sleep(retryTimeout)
			continue
		}
		if latestRelease.Name != currentVersion {
			notifyUpdateAvailable(log, latestRelease)
		}
		return
	}
}

func fetch(client *http.Client) (githubLatestRelease, error) {
	var latestRelease githubLatestRelease
	resp, err := client.Get(latestVersionApiEndpoint)
	if err != nil {
		return latestRelease, err
	}
	if resp.StatusCode != http.StatusOK {
		return latestRelease, fmt.Errorf("version status code is %q", resp.Status)
	}
	if err = json.NewDecoder(resp.Body).Decode(&latestRelease); err != nil {
		return latestRelease, fmt.Errorf("error decoding lastest version response: %v", err)
	}

	return latestRelease, nil
}

func notifyUpdateAvailable(log logger.Logger, latestRelease githubLatestRelease) {
	log.Raw("")
	log.Raw("########################################################################################################################")
	log.Rawf("New version available: %s", latestRelease.Name)
	log.Rawf("Release notes: %s", latestRelease.Body)
	log.Rawf("Download link: %s", latestRelease.HtmlUrl)
	log.Raw("########################################################################################################################")
	log.Raw("")
}
