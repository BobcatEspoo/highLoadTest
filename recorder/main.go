package main

import (
	"flag"
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

func main() {
	name := flag.String("name", "0", "to add name to file")
	flag.Parse()
	fmt.Println("🚀 Start recording on 5 minutes...")
	timestamp := time.Now().Format("150405")
	outputFile := filepath.Join(fmt.Sprintf("recording_%s_%s.mp4", timestamp, *name))

	duration := 5 * time.Minute

	err := recordScreen(outputFile, duration)
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
