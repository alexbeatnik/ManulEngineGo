//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alexbeatnik/ManulHeart/pkg/cdp"
	"github.com/alexbeatnik/ManulHeart/pkg/heuristics"
)

func main() {
	ctx := context.Background()
	conn, err := cdp.NewConn("http://127.0.0.1:9222")
	if err != nil {
		panic(err)
	}

	page, err := cdp.NewPage(ctx, conn, "test")
	if err != nil {
		panic(err)
	}

	val, err := page.CallProbe(ctx, heuristics.BuildExtractProbe(), []string{"CPU of Chrome", ""})
	fmt.Printf("Val: %s Error: %v\n", val, err)
}
