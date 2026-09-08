package hypevidence

import (
	"bytes"
	"errors"
	stdhtml "html"
	"io"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

var excludedHTMLNodes = map[string]bool{"script": true, "style": true, "noscript": true, "svg": true, "canvas": true}

func NormalizeHTML(raw []byte) (string, error) {
	root, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	var b strings.Builder
	var walk func(*html.Node, bool)
	walk = func(node *html.Node, excluded bool) {
		if node == nil {
			return
		}
		isExcluded := excluded || node.Type == html.ElementNode && excludedHTMLNodes[strings.ToLower(node.Data)]
		if !isExcluded && node.Type == html.TextNode {
			if b.Len() > 0 && !strings.HasSuffix(b.String(), " ") {
				b.WriteByte(' ')
			}
			appendNormalized(&b, stdhtml.UnescapeString(node.Data))
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child, isExcluded)
		}
	}
	walk(root, false)
	return strings.TrimSpace(b.String()), nil
}

func NormalizeText(raw []byte) string {
	var b strings.Builder
	appendNormalized(&b, string(raw))
	return strings.TrimSpace(b.String())
}

func appendNormalized(b *strings.Builder, value string) {
	lastSpace := b.Len() == 0 || strings.HasSuffix(b.String(), " ")
	for _, r := range value {
		if unicode.IsSpace(r) {
			if !lastSpace {
				b.WriteByte(' ')
			}
			lastSpace = true
			continue
		}
		b.WriteRune(r)
		lastSpace = false
	}
}

func EstimateTokens(text string) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	// This is a conservative, deterministic pre-tokenizer estimate used only
	// for budgeting. It is not a provider tokenizer claim.
	return (len([]rune(text)) + 3) / 4
}

func ReadAllBounded(r io.Reader, max int64) ([]byte, error) {
	if max <= 0 {
		return nil, errors.New("positive byte limit is required")
	}
	limited := io.LimitReader(r, max+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, errors.New("document exceeds bounded capture limit")
	}
	return data, nil
}
