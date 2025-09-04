package main

import (
	"log"
	"os/exec"
)

func main() {
	// Путь до chrome.exe (проверьте актуальность пути на вашем ПК)
	// chromePath := `C:\Program Files\Google\Chrome\Application\chrome.exe`
	chromeLinuxPath := "/usr/local/bin/google-chrome"
	url := "https://x.la/cgs/empire-of-the-ants/play"

	// Команда запуска
	cmd := exec.Command(chromeLinuxPath, url)

	// Запускаем процесс
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Ошибка запуска Chrome: %v", err)
	}
}
