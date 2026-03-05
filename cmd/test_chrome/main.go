package main

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"time"
)

func main() {
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(
		context.Background(),
		"ws://localhost:9222",
	)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var title string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://hokinhdoanh.dkkd.gov.vn/HkdOnline/Default.aspx"),
		chromedp.WaitReady("body"),
		chromedp.Title(&title),
	)

	if err != nil {
		fmt.Println("❌ Lỗi:", err)
		return
	}

	fmt.Println("✅ Kết nối thành công!")
	fmt.Println("📄 Title:", title)
}
