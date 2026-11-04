package report

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// Export prints local HTML as PDF and captures event groups as separate PNGs.
// Existing report PNGs are removed only after every new output has been rendered.
func Export(ctx context.Context, htmlPath, outDir, browserPath string, perPage int) error {
	if perPage <= 0 {
		return fmt.Errorf("1ページの件数は正の整数で指定してください")
	}
	absolute, err := filepath.Abs(htmlPath)
	if err != nil {
		return err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return fmt.Errorf("HTMLを開く: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("HTMLは通常のファイルで指定してください")
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(outDir, ".report-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if browserPath != "" {
		options = append(options, chromedp.ExecPath(browserPath))
	}
	allocator, cancel := chromedp.NewExecAllocator(ctx, options...)
	defer cancel()
	browser, closeBrowser := chromedp.NewContext(allocator)
	defer closeBrowser()
	localURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}).String()
	var pdf []byte
	var count int
	_, err = chromedp.Run(browser, func(ctx context.Context, t *chromedp.Target) (chromedp.Void, error) {
		if _, e := cdp.Call(ctx, t, network.Enable, network.EnableParams{}); e != nil {
			return chromedp.Void{}, e
		}
		if _, e := cdp.Call(ctx, t, network.SetBlockedURLs, network.SetBlockedURLsParams{URLPatterns: []*network.BlockPattern{{URLPattern: "http://*/*", Block: true}, {URLPattern: "https://*/*", Block: true}, {URLPattern: "ftp://*/*", Block: true}, {URLPattern: "ws://*/*", Block: true}, {URLPattern: "wss://*/*", Block: true}}}); e != nil {
			return chromedp.Void{}, e
		}
		if _, e := cdp.Call(ctx, t, emulation.SetDeviceMetricsOverride, emulation.SetDeviceMetricsOverrideParams{Width: 850, Height: 1000, DeviceScaleFactor: 1, Mobile: false}); e != nil {
			return chromedp.Void{}, e
		}
		if _, e := chromedp.Navigate(localURL)(ctx, t); e != nil {
			return chromedp.Void{}, e
		}
		if _, e := chromedp.Evaluate[bool](`document.fonts.ready.then(()=>true)`, chromedp.EvalAwaitPromise)(ctx, t); e != nil {
			return chromedp.Void{}, e
		}
		var e error
		count, e = chromedp.Evaluate[int](`document.querySelectorAll('[data-event]').length`)(ctx, t)
		if e != nil {
			return chromedp.Void{}, e
		}
		yes := true
		result, e := cdp.Call(ctx, t, page.PrintToPDF, page.PrintToPDFParams{PrintBackground: &yes, PreferCSSPageSize: &yes})
		pdf = result.Data
		return chromedp.Void{}, e
	})
	if err != nil {
		return fmt.Errorf("ブラウザでPDFを生成: %w", err)
	}
	if err := os.WriteFile(filepath.Join(staging, "report.pdf"), pdf, 0644); err != nil {
		return err
	}
	pages := (count + perPage - 1) / perPage
	if pages == 0 {
		pages = 1
	}
	for index := 0; index < pages; index++ {
		var png []byte
		script := fmt.Sprintf(`Array.from(document.querySelectorAll('[data-event]')).forEach((el,i)=>{el.style.display=(i >= %d && i < %d)?'':'none'})`, index*perPage, (index+1)*perPage)
		if _, err := chromedp.Run(browser, chromedp.Evaluate[chromedp.Void](script)); err != nil {
			return err
		}
		var captureErr error
		png, captureErr = chromedp.Run(browser, chromedp.FullScreenshot(100))
		if captureErr != nil {
			return fmt.Errorf("PNG %d を生成: %w", index+1, captureErr)
		}
		if err := os.WriteFile(filepath.Join(staging, fmt.Sprintf("report-%03d.png", index+1)), png, 0644); err != nil {
			return err
		}
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`^report-[0-9]+\.png$`)
	for _, entry := range entries {
		if !entry.IsDir() && pattern.MatchString(entry.Name()) {
			if err := os.Remove(filepath.Join(outDir, entry.Name())); err != nil {
				return err
			}
		}
	}
	generated, err := os.ReadDir(staging)
	if err != nil {
		return err
	}
	for _, entry := range generated {
		if err := os.Rename(filepath.Join(staging, entry.Name()), filepath.Join(outDir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
