package hypevidence

import "testing"

func TestParseIndexAndSelectsPrimaryAndTextDocuments(t *testing.T) {
	raw := []byte(`<table><tr><th>Seq</th><th>Description</th><th>Document</th><th>Type</th></tr><tr><td>1</td><td>8-K</td><td><a href="/ix?doc=/Archives/edgar/data/1/abc/main.htm">main.htm</a></td><td>8-K</td></tr><tr><td>2</td><td>Press release</td><td><a href="/Archives/edgar/data/1/abc/ex99.htm">ex99.htm</a></td><td>EX-99.1</td></tr><tr><td>3</td><td>Image</td><td><a href="/Archives/edgar/data/1/abc/logo.png">logo.png</a></td><td>GRAPHIC</td></tr></table>`)
	index, err := ParseIndex(raw)
	if err != nil {
		t.Fatal(err)
	}
	selected := SelectTextDocuments(index, "main.htm")
	if len(index) != 3 || len(selected) != 2 || selected[0].Filename != "main.htm" || selected[1].Filename != "ex99.htm" {
		t.Fatalf("index=%+v selected=%+v", index, selected)
	}
}

func TestArchiveURLRejectsExternalDocument(t *testing.T) {
	index, err := ArchiveIndexURL("0000320193", "0000320193-24-000001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ArchiveDocumentURL(index, "https://evil.example/document.htm"); err == nil {
		t.Fatal("external document URL accepted")
	}
}

func TestSelectTextDocumentsExcludesDuplicateSubmissionText(t *testing.T) {
	index := []IndexDocument{
		{Filename: "main.htm", Href: "/Archives/edgar/data/1/a/main.htm"},
		{Filename: "0000002488-20-000006.txt", Href: "/Archives/edgar/data/1/a/0000002488-20-000006.txt"},
		{Filename: "ex99.htm", Href: "/Archives/edgar/data/1/a/ex99.htm"},
	}
	selected := SelectTextDocuments(index, "main.htm")
	if len(selected) != 2 || selected[0].Filename != "main.htm" || selected[1].Filename != "ex99.htm" {
		t.Fatalf("selected duplicate submission text: %+v", selected)
	}
}
