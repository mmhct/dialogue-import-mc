//go:build !windows

package app

import (
	"fmt"
	"log"
)

func openWindowsBrowser(url string) error { return fmt.Errorf("Windows browser unavailable") }
func NotifyError(message string)          { log.Print(message) }
