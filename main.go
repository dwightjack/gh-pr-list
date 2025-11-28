package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/template"
	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/yarlson/pin"
)

type SearchResponse struct {
	Items []struct {
		Title  string `json:"title"`
		URL    string `json:"html_url"`
		Number int    `json:"number"`
	} `json:"items"`
}

type Config struct {
	org        string
	asMarkdown bool
	asJSON     bool
}

const lineTemplate = `{{range .}}{{hyperlink .html_url (printf "#%.0f - %s" .number .title)}}
{{end}}`

func parseFlags() Config {
	var org string
	var asMarkdown bool
	var asJSON bool
	flag.StringVar(&org, "org", "", "The organization to search for PRs")
	flag.BoolVar(&asMarkdown, "markdown", false, "Output as markdown")
	flag.BoolVar(&asJSON, "json", false, "Output as JSON")
	flag.Parse()

	if asMarkdown && asJSON {
		log.Fatal("Cannot use both markdown and JSON output")
	}

	return Config{
		org:        org,
		asMarkdown: asMarkdown,
		asJSON:     asJSON,
	}

}

func fetchPRs(client *api.RESTClient, org string) (SearchResponse, error) {
	searchQuery := "search/issues?q=is:pr+is:open+author:@me"
	if org != "" {
		searchQuery += "+org:" + url.QueryEscape(org)
	}

	response := SearchResponse{}
	apiError := client.Get(searchQuery, &response)
	return response, apiError
}

func renderTerminal(templateString string, response SearchResponse) (string, error) {
	t := term.FromEnv()
	width, _, err := t.Size()
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(response.Items)
	if err != nil {
		return "", err
	}

	var str strings.Builder
	tmpl := template.New(&str, width, t.IsColorEnabled())

	if err := tmpl.Parse(templateString); err != nil {
		return "", err
	}

	if err := tmpl.Execute(bytes.NewReader(data)); err != nil {
		return "", err
	}

	return str.String(), nil
}

func run() (string, error) {
	cfg := parseFlags()
	client, err := api.DefaultRESTClient()
	if err != nil {
		return "", err
	}
	response, err := fetchPRs(client, cfg.org)
	if err != nil {
		return "", nil
	}

	if len(response.Items) == 0 {
		fmt.Fprintln(os.Stderr, "No PRs found.")
		return "", nil
	}

	if cfg.asMarkdown {
		var str strings.Builder
		for _, item := range response.Items {
			str.WriteString(fmt.Sprintf("* [#%d - %s](%s)\n", item.Number, item.Title, item.URL))
		}
		return str.String(), nil
	}
	if cfg.asJSON {
		json, err := json.MarshalIndent(response.Items, "", "  ")
		return string(json), err
	}

	return renderTerminal(lineTemplate, response)

}

func main() {

	p := pin.New("Fetching PRs...", pin.WithWriter(os.Stderr))
	cancel := p.Start(context.Background())
	defer cancel()
	str, err := run()

	p.Stop()

	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(str)
}
