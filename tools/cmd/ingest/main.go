package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"
)

type FrontMatter struct {
	Title      string   `yaml:"title"`
	Version    string   `yaml:"version"`
	Status     string   `yaml:"status"`
	CreatedUTC string   `yaml:"created_utc"`
	Tags       []string `yaml:"tags"`
}

type Doc struct {
	DocType    string
	RelPath    string
	Title      string
	Version    string
	Status     string
	CreatedUTC time.Time
	UpdatedUTC time.Time
	TagsJSON   []byte
	Sha256     string
	Markdown   string
}

func main() {
	var root string
	var dsn string
	var dryRun bool

	flag.StringVar(&root, "root", ".", "Root folder containing markdown files")
	flag.StringVar(&dsn, "dsn", "", "PostgreSQL DSN, e.g. postgres://user:pass@host:5432/db?sslmode=disable")
	flag.BoolVar(&dryRun, "dry-run", true, "If true, does not write to DB")
	flag.Parse()

	if dsn == "" {
		fmt.Println("Missing --dsn")
		os.Exit(2)
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	// Folder prefix -> doc_type mapping (simple & effective)
	prefixToType := map[string]string{
		"strategies/known":      "strategy",
		"strategies/discovered": "strategy",
		"patterns":              "pattern",
		"anti-patterns":         "anti-pattern",
		"meta":                  "meta",
		"risk":                  "risk",
		"evaluation":            "evaluation",
	}

	var files []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found %d markdown files\n", len(files))

	changed := 0
	skipped := 0
	for _, file := range files {
		mdBytes, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("ERROR reading %s: %v\n", file, err)
			continue
		}
		md := string(mdBytes)

		sha := sha256Hex(md)
		rel := toRel(root, file)
		docType := resolveType(rel, prefixToType)

		existingSha, err := getShaByRelPath(ctx, pool, rel)
		if err != nil {
			fmt.Printf("ERROR sha lookup %s: %v\n", rel, err)
			continue
		}
		if existingSha != "" && existingSha == sha {
			skipped++
			continue
		}

		fm, _ := parseFrontMatter(md)
		created := parseCreatedUTC(fm.CreatedUTC)

		// defaults/hardening
		title := strings.TrimSpace(fm.Title)
		if title == "" {
			title = "Untitled"
		}
		version := strings.TrimSpace(fm.Version)
		if version == "" {
			version = "0.0"
		}
		status := strings.TrimSpace(fm.Status)
		if status == "" {
			status = "draft"
		}
		if fm.Tags == nil {
			fm.Tags = []string{}
		}
		tagsJSON, _ := json.Marshal(fm.Tags)

		doc := Doc{
			DocType:    docType,
			RelPath:    rel,
			Title:      title,
			Version:    version,
			Status:     status,
			CreatedUTC: created,
			UpdatedUTC: time.Now().UTC(),
			TagsJSON:   tagsJSON,
			Sha256:     sha,
			Markdown:   md,
		}

		fmt.Printf("Upsert %s (%s) title=%q version=%s status=%s\n", doc.RelPath, doc.DocType, doc.Title, doc.Version, doc.Status)

		if !dryRun {
			if err := upsertDoc(ctx, pool, doc); err != nil {
				fmt.Printf("ERROR upsert %s: %v\n", doc.RelPath, err)
				continue
			}
		}
		changed++
	}

	fmt.Printf("Done. changed=%d skipped=%d dryRun=%v\n", changed, skipped, dryRun)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func toRel(root, full string) string {
	rel, err := filepath.Rel(root, full)
	if err != nil {
		return full
	}
	return filepath.ToSlash(rel)
}

func resolveType(rel string, m map[string]string) string {
	rel = filepath.ToSlash(rel)
	bestKey := ""
	bestType := "unknown"
	for k, v := range m {
		if strings.HasPrefix(strings.ToLower(rel), strings.ToLower(filepath.ToSlash(k))) {
			if len(k) > len(bestKey) {
				bestKey = k
				bestType = v
			}
		}
	}
	return bestType
}

func parseFrontMatter(md string) (FrontMatter, string) {
	// YAML front matter: starts with --- and ends with --- on its own line
	var fm FrontMatter
	if !strings.HasPrefix(md, "---") {
		return FrontMatter{}, md
	}
	parts := strings.SplitN(md, "\n---", 2)
	if len(parts) < 2 {
		return FrontMatter{}, md
	}
	yamlBlock := strings.TrimPrefix(parts[0], "---")
	body := strings.TrimPrefix(parts[1], "\n")

	_ = yaml.Unmarshal([]byte(yamlBlock), &fm)
	return fm, body
}

func parseCreatedUTC(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Now().UTC()
	}
	// Accept YYYY-MM-DD or RFC3339
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC()
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC()
	}
	return time.Now().UTC()
}

func getShaByRelPath(ctx context.Context, pool *pgxpool.Pool, rel string) (string, error) {
	const q = `SELECT sha256 FROM strategy_documents WHERE rel_path=$1`
	var sha string
	err := pool.QueryRow(ctx, q, rel).Scan(&sha)
	if err != nil {
		// not found
		return "", nil
	}
	return sha, nil
}

func upsertDoc(ctx context.Context, pool *pgxpool.Pool, d Doc) error {
	const q = `
INSERT INTO strategy_documents (doc_type, rel_path, title, version, status, created_utc, updated_utc, tags, sha256, markdown)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10)
ON CONFLICT (rel_path) DO UPDATE SET
  doc_type=EXCLUDED.doc_type,
  title=EXCLUDED.title,
  version=EXCLUDED.version,
  status=EXCLUDED.status,
  created_utc=EXCLUDED.created_utc,
  updated_utc=EXCLUDED.updated_utc,
  tags=EXCLUDED.tags,
  sha256=EXCLUDED.sha256,
  markdown=EXCLUDED.markdown;
`
	_, err := pool.Exec(ctx, q,
		d.DocType, d.RelPath, d.Title, d.Version, d.Status,
		d.CreatedUTC, d.UpdatedUTC, string(d.TagsJSON), d.Sha256, d.Markdown,
	)
	return err
}
