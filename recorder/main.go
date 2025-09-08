package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func getDisplayNumber() string {
	if os.Getenv("DISPLAY") != "" {
		return os.Getenv("DISPLAY")
	}

	cmd := exec.Command("who", "|", "grep", "(:")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {

		return ":0"
	}

	return ":0"
}

func recordScreen(outputPath string, duration time.Duration) error {
	display := getDisplayNumber()

	cmd := exec.Command("ffmpeg",
		"-f", "x11grab", // Формат захвата X11
		"-video_size", "1920x1080", // Разрешение
		"-framerate", "24", // Частота кадров
		"-i", display, // Дисплей для захвата
		"-t", fmt.Sprintf("%.0f", duration.Seconds()), // Длительность
		"-pix_fmt", "yuv420p", // Формат пикселей
		"-c:v", "libx264", // Кодек видео
		"-preset", "fast", // Пресет кодирования
		"-crf", "35", // Качество (0-51, меньше - лучше)
		outputPath, // Выходной файл
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Start recording %v...\n", duration)
	fmt.Printf("Comand: %s\n", cmd.String())

	return cmd.Run()
}

func createOutputDir() (string, error) {
	dirName := fmt.Sprintf("recordings_%s", time.Now().Format("20060102_150405"))
	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		return "", fmt.Errorf("error with directory: %v", err)
	}
	return dirName, nil
}

func main() {
	fmt.Println("🚀 Start recording on 5 minutes...")

	outputDir, err := createOutputDir()
	if err != nil {
		log.Fatalf("❌ Ошибка: %v", err)
	}

	timestamp := time.Now().Format("150405")
	outputFile := filepath.Join(outputDir, fmt.Sprintf("recording_%s.mp4", timestamp))

	duration := 5 * time.Minute

	err = recordScreen(outputFile, duration)
	if err != nil {
		log.Fatalf("❌ Error record: %v", err)
	}

	fmt.Printf("✅ Record done! File save: %s\n", outputFile)
	fmt.Printf("📊 File size: ")

	if info, err := os.Stat(outputFile); err == nil {
		sizeMB := float64(info.Size()) / (1024 * 1024)
		fmt.Printf("%.2f MB\n", sizeMB)
	}
}
