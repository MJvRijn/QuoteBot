package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func getQuotesFromGithub() ([]string, error) {
	req, err := http.NewRequest("GET", os.Getenv("GITHUB_QUOTE_FILE"), nil)
	if err != nil {
		return nil, err
	}

	req.Header["Accept"] = []string{"application/vnd.github.raw+json"}
	req.Header["Authorization"] = []string{fmt.Sprintf("Bearer %s", os.Getenv("GITHUB_ACCESS_TOKEN"))}
	req.Header["X-GitHub-Api-Version"] = []string{"2022-11-28"}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(body), "\n"), nil
}
