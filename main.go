package main

import (
	"log"
	"os/exec"
)

func main() {
	// Путь до chrome.exe (проверьте актуальность пути на вашем ПК)
	chromePath := `C:\Program Files\Google\Chrome\Application\chrome.exe`
	url := "https://x.la/cgs/empire-of-the-ants/play"

	// Команда запуска
	cmd := exec.Command(chromePath, url)

	// Запускаем процесс
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Ошибка запуска Chrome: %v", err)
	}
}
