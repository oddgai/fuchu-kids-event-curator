// Package app contains the command-line workflow.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/oddgai/fuchu-kids-event-curator/internal/collect"
	"github.com/oddgai/fuchu-kids-event-curator/internal/event"
	"github.com/oddgai/fuchu-kids-event-curator/internal/report"
)

type Config struct {
	AgeMonths  int      `json:"age_months"`
	Days       int      `json:"days"`
	Sources    []string `json:"sources"`
	MaxPages   int      `json:"max_pages"`
	MaxEvents  int      `json:"max_events_per_source"`
	IntervalMS int      `json:"request_interval_ms"`
	PerPage    int      `json:"image_events_per_page"`
}

func defaults() Config {
	return Config{AgeMonths: 32, Days: 30, MaxPages: 2, MaxEvents: 30, IntervalMS: 1000, PerPage: 5}
}
func loadConfig(path string) (Config, error) {
	c := defaults()
	if path == "" {
		return c, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return c, fmt.Errorf("設定JSONの末尾に余分なデータがあります")
	}
	return c, nil
}
func Run(ctx context.Context, args []string, out io.Writer) error {
	// Read config before defining flags, so explicit flags override its values.
	configPath := ""
	for i, a := range args {
		if a == "-config" || a == "--config" {
			if i+1 < len(args) {
				configPath = args[i+1]
			}
		}
		if strings.HasPrefix(a, "-config=") {
			configPath = strings.TrimPrefix(a, "-config=")
		}
		if strings.HasPrefix(a, "--config=") {
			configPath = strings.TrimPrefix(a, "--config=")
		}
	}
	c, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("設定: %w", err)
	}
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	now := time.Now().In(jst)
	fs := flag.NewFlagSet("fuchu-kids-event-curator", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&configPath, "config", configPath, "設定JSON（省略時は既定値）")
	from := fs.String("from", now.Format("2006-01-02"), "掲載開始日 YYYY-MM-DD（日本時間）")
	fs.IntVar(&c.Days, "days", c.Days, "掲載日数（開始日を含む、最大366）")
	fs.IntVar(&c.AgeMonths, "age-months", c.AgeMonths, "子供の月齢")
	sourceIDs := fs.String("sources", strings.Join(c.Sources, ","), "収集元IDをカンマ区切り（省略時全情報源）")
	fs.IntVar(&c.MaxPages, "max-pages", c.MaxPages, "1情報源の一覧取得上限")
	fs.IntVar(&c.MaxEvents, "max-events", c.MaxEvents, "1情報源の詳細取得上限")
	fs.IntVar(&c.PerPage, "per-page", c.PerPage, "PNG1枚あたりのイベント件数")
	output := fs.String("output", "output", "成果物の保存先")
	input := fs.String("input", "", "保存済み収集JSONから再生成（通信しない）")
	export := fs.Bool("export", false, "PDFと分割PNGも生成する（Chromiumが必要）")
	browser := fs.String("browser", os.Getenv("CHROME_PATH"), "Chrome/Chromiumの実行ファイル")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("不明な引数: %s", strings.Join(fs.Args(), " "))
	}
	if c.Days < 1 || c.Days > 366 || c.AgeMonths < 0 || c.AgeMonths > 216 || c.MaxPages < 1 || c.MaxPages > 12 || c.MaxEvents < 1 || c.MaxEvents > 200 || c.IntervalMS < 0 || c.PerPage < 1 || c.PerPage > 30 {
		return fmt.Errorf("日数・月齢・取得上限・間隔・PNG件数の設定値が範囲外です")
	}
	start, err := time.ParseInLocation("2006-01-02", *from, jst)
	if err != nil {
		return fmt.Errorf("開始日: %w", err)
	}
	until := start.AddDate(0, 0, c.Days-1).Format("2006-01-02")
	var ids []string
	valid := map[string]bool{}
	for _, s := range collect.Sources() {
		valid[s.ID] = true
	}
	if *sourceIDs != "" {
		for _, id := range strings.Split(*sourceIDs, ",") {
			id = strings.TrimSpace(id)
			if !valid[id] {
				return fmt.Errorf("不明な情報源ID: %q", id)
			}
			ids = append(ids, id)
		}
	}
	raw := event.Report{GeneratedAt: now, From: *from, Until: until, AgeMonths: c.AgeMonths}
	if *input != "" {
		f, err := os.Open(*input)
		if err != nil {
			return err
		}
		err = json.NewDecoder(io.LimitReader(f, 32<<20)).Decode(&raw)
		f.Close()
		if err != nil {
			return fmt.Errorf("保存JSON: %w", err)
		}
	} else {
		raw.Events, raw.Sources = collect.Run(ctx, collect.NewClient(time.Duration(c.IntervalMS)*time.Millisecond), collect.Options{IDs: ids, MaxPages: c.MaxPages, MaxEvents: c.MaxEvents, From: *from, Until: until, BrowserPath: *browser})
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(*output, "collected.json"), raw); err != nil {
		return err
	}
	result := event.Report{GeneratedAt: now, From: *from, Until: until, AgeMonths: c.AgeMonths, Sources: raw.Sources, Events: event.Select(raw.Events, *from, until, c.AgeMonths)}
	sort.SliceStable(result.Events, func(i, j int) bool {
		a, b := result.Events[i], result.Events[j]
		if len(a.Dates) == 0 {
			return false
		}
		if len(b.Dates) == 0 {
			return true
		}
		return a.Dates[0].Start < b.Dates[0].Start
	})
	successes := 0
	for _, s := range result.Sources {
		if s.Status == "success" || s.Status == "partial" {
			successes++
		}
		if s.Status != "success" {
			result.Warnings = append(result.Warnings, s.Name+": "+s.Error)
		}
	}
	result.Warnings = append(result.Warnings, "年齢や好みによるおすすめは補助情報です。申込・お出かけ前に出典で最新の参加条件と開催状況をご確認ください。")
	if err := writeJSON(filepath.Join(*output, "report.json"), result); err != nil {
		return err
	}
	htmlPath := filepath.Join(*output, "report.html")
	f, err := os.Create(htmlPath)
	if err != nil {
		return err
	}
	err = report.WriteHTML(f, result)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if *export {
		browserCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		defer cancel()
		if err := report.Export(browserCtx, htmlPath, *output, *browser, c.PerPage); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "候補 %d件 / 収集 %d件 → %s\n", len(result.Events), len(raw.Events), htmlPath)
	for _, s := range result.Sources {
		fmt.Fprintf(out, "%s: %s (%d件) %s\n", s.ID, s.Status, s.Count, s.Error)
	}
	if successes == 0 {
		return fmt.Errorf("すべての情報源で取得失敗または無効です。状況を記載した成果物は保存しました")
	}
	return nil
}
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}
