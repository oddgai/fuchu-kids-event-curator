package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: fuchu-kids-event-curator [-help]")
		fmt.Fprintln(os.Stderr, "府中市と近郊の子供向けイベントをまとめるCLIです。")
		fmt.Fprintln(os.Stderr, "現在は開発環境の骨格のみです。収集・レポート生成は未実装です。")
	}
	flag.Parse()
	flag.Usage()
}
