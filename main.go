package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/template"
	"github.com/cli/go-gh/v2/pkg/term"
)

type SearchResponse struct {
	Items []struct {
		Title string `json:"title"`
		URL   string `json:"html_url"`
		ID    int    `json:"number"`
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

func renderTerminal(templateString string, response SearchResponse) error {
	t := term.FromEnv()
	width, _, err := t.Size()
	if err != nil {
		return err
	}

	tmpl := template.New(os.Stdout, width, t.IsColorEnabled())
	data, err := json.Marshal(response.Items)
	if err != nil {
		return err
	}

	if err := tmpl.Parse(templateString); err != nil {
		return err
	}
	if err := tmpl.Execute(bytes.NewReader(data)); err != nil {
		return err
	}
	return nil
}

func run() error {
	cfg := parseFlags()
	client, err := api.DefaultRESTClient()
	if err != nil {
		return err
	}
	response, err := fetchPRs(client, cfg.org)
	if err != nil {
		return err
	}

	if len(response.Items) == 0 {
		fmt.Fprintln(os.Stderr, "No PRs found.")
		return nil
	}

	if cfg.asMarkdown {
		for _, item := range response.Items {
			fmt.Printf("* [#%d - %q](%q)\n", item.ID, item.Title, item.URL)
		}
		return nil
	}
	if cfg.asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(response.Items)
	}

	return renderTerminal(lineTemplate, response)

}

func main() {

	if err := run(); err != nil {
		log.Fatal(err)
	}
}
