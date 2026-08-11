//go:build windows

package main

import (
	"fmt"
	"os"

	"github.com/DylonH78/AppAlias/internal/service"
	"github.com/DylonH78/AppAlias/internal/ui"
)

func main() {
	svc, err := service.New("", false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "AppAlias:", err)
		os.Exit(1)
	}
	ui.Show(svc)
}
