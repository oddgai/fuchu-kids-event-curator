package report

import (
	"bytes"
	"context"
	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestExportRejectsInvalidInputBeforeStartingBrowser(t *testing.T) {
	if err := Export(context.Background(), "missing.html", t.TempDir(), "", 0); err == nil {
		t.Fatal("non-positive page size accepted")
	}
	if err := Export(context.Background(), "missing.html", t.TempDir(), "", 5); err == nil {
		t.Fatal("missing HTML accepted")
	}
}
func TestExportBrowser(t *testing.T) {
	browser := os.Getenv("REPORT_TEST_BROWSER")
	if browser == "" {
		t.Skip("set REPORT_TEST_BROWSER for browser integration")
	}
	for _, count := range []int{0, 1, 13} {
		t.Run(string(rune('A'+count)), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "input.html")
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			r := browserReport(count)
			if err := WriteHTML(f, r); err != nil {
				t.Fatal(err)
			}
			f.Close()
			os.WriteFile(filepath.Join(dir, "report-099.png"), []byte("old"), 0600)
			if err := Export(context.Background(), path, dir, browser, 5); err != nil {
				t.Fatal(err)
			}
			pdf, err := os.ReadFile(filepath.Join(dir, "report.pdf"))
			if err != nil || len(pdf) < 100 || string(pdf[:4]) != "%PDF" {
				t.Fatalf("invalid PDF: %v", err)
			}
			files, err := filepath.Glob(filepath.Join(dir, "report-*.png"))
			if err != nil {
				t.Fatal(err)
			}
			want := (count + 4) / 5
			if want == 0 {
				want = 1
			}
			if len(files) != want {
				t.Fatalf("PNG count=%d, want %d", len(files), want)
			}
			for _, path := range files {
				data, err := os.ReadFile(path)
				if err != nil || len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
					t.Fatalf("invalid PNG %s", path)
				}
				if count == 13 && filepath.Base(path) == "report-001.png" {
					img, err := png.DecodeConfig(bytes.NewReader(data))
					if err != nil {
						t.Fatal(err)
					}
					if img.Height < 2000 {
						t.Fatalf("PNG clipped at height %d", img.Height)
					}
				}
			}
		})
	}
}

func browserReport(count int) event.Report {
	r := event.Report{}
	for i := 0; i < count; i++ {
		r.Events = append(r.Events, event.Event{Title: "親子で楽しむイベント", Summary: "長い日本語の案内です。自然に触れて楽しみましょう。"})
	}
	return r
}
