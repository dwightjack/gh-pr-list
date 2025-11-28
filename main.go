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
	graphql "github.com/cli/shurcooL-graphql"
	"github.com/yarlson/pin"
)

type ItemRepository struct {
	NameWithOwner string `json:"nameWithOwner"`
	Name          string `json:"name"`
	Url           string `json:"url"`
}

type Item struct {
	Title      string         `json:"title"`
	Url        string         `json:"url"`
	Number     int            `json:"number"`
	Repository ItemRepository `json:"repository"`
}

type ItemListByRepo map[string][]*Item

type Config struct {
	org        string
	asMarkdown bool
	asJSON     bool
	limit      int
}

const terminalTemplate = `{{range $repo, $items := .}}{{$repo}}
	{{range $items}}
  {{hyperlink .url (printf "#%.0f - %s" .number .title)}}
	{{end}}
{{end}}`

func parseFlags() Config {
	var org string
	var asMarkdown bool
	var asJSON bool
	var limit int
	flag.StringVar(&org, "org", "", "The organization to search for PRs")
	flag.BoolVar(&asMarkdown, "markdown", false, "Output as markdown")
	flag.BoolVar(&asJSON, "json", false, "Output as JSON")
	flag.IntVar(&limit, "limit", 10, "The max number of results")
	flag.Parse()

	if asMarkdown && asJSON {
		log.Fatal("Cannot use both markdown and JSON output")
	}

	return Config{
		org:        org,
		asMarkdown: asMarkdown,
		asJSON:     asJSON,
		limit:      limit,
	}

}

func fetchPRs(client *api.GraphQLClient, org string, limit int) (int, ItemListByRepo, error) {
	searchQuery := "is:pr is:open author:@me"
	if org != "" {
		searchQuery += " org:" + url.QueryEscape(org)
	}

	var query struct {
		Search struct {
			Nodes []struct {
				PullRequest struct {
					Number     int
					Url        string
					Title      string
					Repository ItemRepository
				} `graphql:"... on PullRequest"`
			}
		} `graphql:"search(query: $query, type: ISSUE, first: $first)"`
	}

	variables := map[string]any{
		"query": graphql.String(searchQuery),
		"first": graphql.Int(min(limit, 100)),
	}

	apiError := client.Query("Search", &query, variables)
	count := len(query.Search.Nodes)
	itemList := make(ItemListByRepo)

	for _, item := range query.Search.Nodes {
		key := item.PullRequest.Repository.NameWithOwner

		itemList[key] = append(itemList[key], &Item{
			Title:      item.PullRequest.Title,
			Url:        item.PullRequest.Url,
			Number:     item.PullRequest.Number,
			Repository: item.PullRequest.Repository,
		})
	}
	return count, itemList, apiError
}

func renderTerminal(templateString string, itemList ItemListByRepo) (string, error) {
	t := term.FromEnv()
	width, _, err := t.Size()
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(itemList)
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
	client, err := api.NewGraphQLClient(api.ClientOptions{})
	if err != nil {
		return "", err
	}
	count, itemList, err := fetchPRs(client, cfg.org, cfg.limit)
	if err != nil {
		return "", nil
	}

	if count == 0 {
		fmt.Fprintln(os.Stderr, "No PRs found.")
		return "", nil
	}

	if cfg.asMarkdown {
		var str strings.Builder
		for key, items := range itemList {
			str.WriteString(fmt.Sprintf("* **%s**\n", key))

			for _, item := range items {
				str.WriteString(fmt.Sprintf(". * [#%d - %s](%s)\n", item.Number, item.Title, item.Url))
			}
		}
		return str.String(), nil
	}
	if cfg.asJSON {
		json, err := json.MarshalIndent(itemList, "", "  ")
		return string(json), err
	}

	return renderTerminal(terminalTemplate, itemList)

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
