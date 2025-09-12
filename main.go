package main

import (
	"log"
	"os/exec"
)

func main() {
	// chromePath := `C:\Program Files\Google\Chrome\Application\chrome.exe`
	chromeLinuxPath := "/usr/local/bin/google-chrome"
	url := "https://x.la/cgs/1754888695/play"
	args := []string{
		"--password-store=basic",
		"--no-first-run",
		"--disable-features=AccountConsistency",
		"--disable-signin-promo",
		url,
	}

	cmd := exec.Command(chromeLinuxPath, args...)

	err := cmd.Start()
	if err != nil {
		log.Fatalf("Error while start chrome: %v", err)
	}
}
