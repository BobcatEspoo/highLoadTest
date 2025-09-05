package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// recordScreen записывает экран с помощью ffmpeg
func recordScreen(outputPath string, duration time.Duration) error {
	display := ":0"

	// Команда ffmpeg для записи экрана
	cmd := exec.Command("ffmpeg",
		"-f", "x11grab", // Формат захвата X11
		"-video_size", "1920x1080", // Разрешение
		"-framerate", "24", // Частота кадров
		"-i", display, // Дисплей для захвата
		"-t", fmt.Sprintf("%.0f", duration.Seconds()), // Длительность
		"-pix_fmt", "yuv420p", // Формат пикселей
		"-c:v", "libx264", // Кодек видео
		"-preset", "fast", // Пресет кодирования
		"-crf", "25", // Качество (0-51, меньше - лучше)
		outputPath, // Выходной файл
	)

	// Направляем вывод в консоль
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Начинаем запись экрана на %v...\n", duration)
	fmt.Printf("Команда: %s\n", cmd.String())

	return cmd.Run()
}

// createOutputDir создает директорию для записи
func createOutputDir() (string, error) {
	dirName := fmt.Sprintf("recordings_%s", time.Now().Format("20060102_150405"))
	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		return "", fmt.Errorf("ошибка создания директории: %v", err)
	}
	return dirName, nil
}

func main() {
	fmt.Println("🚀 Запуск записи экрана на 5 минут...")
	fmt.Println("Для остановки раньше времени нажмите Ctrl+C")

	// Создаем директорию для записи
	outputDir, err := createOutputDir()
	if err != nil {
		log.Fatalf("❌ Ошибка: %v", err)
	}

	// Генерируем имя файла с timestamp
	timestamp := time.Now().Format("150405")
	outputFile := filepath.Join(outputDir, fmt.Sprintf("recording_%s.mp4", timestamp))

	duration := 1 * time.Minute

	// Запускаем запись
	err = recordScreen(outputFile, duration)
	if err != nil {
		log.Fatalf("❌ Ошибка записи: %v", err)
	}

	fmt.Printf("✅ Запись завершена! Файл сохранен: %s\n", outputFile)
	fmt.Printf("📊 Размер файла: ")

	// Показываем размер файла
	if info, err := os.Stat(outputFile); err == nil {
		sizeMB := float64(info.Size()) / (1024 * 1024)
		fmt.Printf("%.2f MB\n", sizeMB)
	}
}
