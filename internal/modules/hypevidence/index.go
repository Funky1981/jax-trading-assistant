package hypevidence

import (
	"bytes"
	"errors"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html"
)

func ParseIndex(raw []byte) ([]IndexDocument, error) {
	root, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	var result []IndexDocument
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}
		if node.Type == html.ElementNode && node.Data == "tr" {
			if doc, ok := parseRow(node); ok {
				result = append(result, doc)
			}
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	if len(result) == 0 {
		return nil, errors.New("SEC filing index contains no document inventory")
	}
	return deduplicateDocuments(result), nil
}

func parseRow(row *html.Node) (IndexDocument, bool) {
	var cells []string
	href := firstHref(row)
	if parsed, err := url.Parse(href); err == nil && parsed.Path == "/ix" {
		// SEC indexes often wrap the primary document in the iX viewer. The
		// accession-bound doc query is still the underlying archive document.
		if document := parsed.Query().Get("doc"); document != "" {
			href = document
		}
	}
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && (node.Data == "td" || node.Data == "th") {
			text := strings.TrimSpace(nodeText(node))
			if text != "" {
				cells = append(cells, text)
			}
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(row)
	if href == "" || !isArchiveReference(href) {
		return IndexDocument{}, false
	}
	parsed, err := url.Parse(href)
	if err != nil || path.Base(parsed.Path) == "" {
		return IndexDocument{}, false
	}
	filename := path.Base(parsed.Path)
	if strings.EqualFold(filename, "-index.html") || strings.HasSuffix(strings.ToLower(filename), "-index.htm") {
		return IndexDocument{}, false
	}
	var description, documentType string
	if len(cells) > 0 {
		description = cells[0]
	}
	if len(cells) > 1 {
		documentType = cells[1]
	}
	return IndexDocument{Filename: filename, DocumentType: documentType, Description: description, Href: href}, true
}

func firstHref(node *html.Node) string {
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if attr.Key == "href" {
				return strings.TrimSpace(attr.Val)
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if href := firstHref(child); href != "" {
			return href
		}
	}
	return ""
}

func nodeText(node *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return b.String()
}

func isArchiveReference(href string) bool {
	parsed, err := url.Parse(href)
	if err != nil {
		return false
	}
	return parsed.Path != "" && (strings.Contains(parsed.Path, "/Archives/edgar/data/") || !parsed.IsAbs() && !strings.HasPrefix(parsed.Path, "/ixviewer"))
}

func deduplicateDocuments(documents []IndexDocument) []IndexDocument {
	seen := make(map[string]bool, len(documents))
	result := make([]IndexDocument, 0, len(documents))
	for _, document := range documents {
		key := strings.ToLower(document.Filename)
		if !seen[key] {
			seen[key] = true
			result = append(result, document)
		}
	}
	return result
}

func SelectTextDocuments(index []IndexDocument, primary string) []IndexDocument {
	primary = strings.ToLower(strings.TrimSpace(primary))
	result := make([]IndexDocument, 0, len(index))
	for _, document := range index {
		filename := strings.ToLower(strings.TrimSpace(document.Filename))
		if filename == primary || isTextFilename(filename) && !strings.Contains(filename, "-index.") && !isDuplicateSubmissionText(filename) {
			result = append(result, document)
		}
	}
	return result
}

func isTextFilename(filename string) bool {
	return strings.HasSuffix(filename, ".htm") || strings.HasSuffix(filename, ".html") || strings.HasSuffix(filename, ".txt")
}

func isDuplicateSubmissionText(filename string) bool {
	base := strings.TrimSuffix(strings.ToLower(filename), ".txt")
	if len(base) != 20 || base[10] != '-' || base[13] != '-' {
		return false
	}
	for index, value := range base {
		if index == 10 || index == 13 {
			continue
		}
		if value < '0' || value > '9' {
			return false
		}
	}
	return true
}
