package app

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOfflineReport(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	b, _ := json.Marshal(event.Report{Events: []event.Event{{Title: "絵本とあそび", URL: "https://example.org/event", Dates: []event.DateRange{{Start: "2026-11-07"}}}}, Sources: []event.SourceResult{{Name: "テスト", Status: "success", Count: 1}}, GeneratedAt: time.Now()})
	os.WriteFile(input, b, 0600)
	var out bytes.Buffer
	if err := Run(context.Background(), []string{"-input", input, "-from", "2026-11-01", "-days", "30", "-output", filepath.Join(dir, "out")}, &out); err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(filepath.Join(dir, "out", "report.html"))
	if err != nil || !strings.Contains(string(html), "絵本とあそび") {
		t.Fatalf("%v %s", err, html)
	}
	if _, err := os.Stat(filepath.Join(dir, "out", "report.json")); err != nil {
		t.Fatal(err)
	}
}
func TestInvalidOptions(t *testing.T) {
	for _, args := range [][]string{{"-days", "0"}, {"-age-months", "-1"}, {"-from", "2026-02-30"}, {"-sources", "typo"}, {"-per-page", "0"}} {
		if err := Run(context.Background(), args, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
func TestAllFailedStillWritesStatus(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "failed.json")
	b, _ := json.Marshal(event.Report{Sources: []event.SourceResult{{Name: "test", Status: "error", Error: "HTTP 500"}}})
	os.WriteFile(p, b, 0600)
	out := filepath.Join(dir, "out")
	err := Run(context.Background(), []string{"-input", p, "-output", out}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("all failed reported success")
	}
	if _, e := os.Stat(filepath.Join(out, "report.html")); e != nil {
		t.Fatal(e)
	}
}

func TestGroupedReportCanBeReclassifiedWithoutLosingSections(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "grouped.json")
	b, _ := json.Marshal(event.Report{Information: []event.Event{{Title: "冬のイルミネーション", Dates: []event.DateRange{{Start: "2026-11-01", End: "2026-12-31"}}}}, Review: []event.Event{{Title: "プレーパーク"}}, Sources: []event.SourceResult{{Status: "success"}}})
	os.WriteFile(input, b, 0600)
	if err := Run(context.Background(), []string{"-input", input, "-from", "2026-11-01", "-output", filepath.Join(dir, "out")}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(dir, "out", "report.json"))
	var got event.Report
	json.Unmarshal(b, &got)
	if len(got.Information) != 1 || len(got.Review) != 1 {
		t.Fatalf("sections lost: %+v", got)
	}
}
