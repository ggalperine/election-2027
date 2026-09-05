package poll

import (
	"encoding/json"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

func decodeJSON(resp *http.Response, v any) error {
	return json.NewDecoder(resp.Body).Decode(v)
}

// findTables returns all <table> nodes with class containing "wikitable".
func findTables(n *html.Node) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "table" {
			if strings.Contains(attr(node, "class"), "wikitable") {
				out = append(out, node)
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

// findRows returns <tr> nodes belonging to the given table (skips nested tables).
func findRows(table *html.Node) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node, bool)
	walk = func(node *html.Node, insideNested bool) {
		if node != table && node.Type == html.ElementNode && node.Data == "table" {
			insideNested = true
		}
		if !insideNested && node.Type == html.ElementNode && node.Data == "tr" {
			out = append(out, node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c, insideNested)
		}
	}
	walk(table, false)
	return out
}

// cellTexts returns the trimmed text of each direct cell (matching one of tags) in a row.
func cellTexts(row *html.Node, tags ...string) []string {
	var out []string
	for c := row.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		for _, t := range tags {
			if c.Data == t {
				out = append(out, strings.TrimSpace(textOf(c)))
				break
			}
		}
	}
	return out
}

func textOf(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
