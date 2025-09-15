package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

func getChromePath() string {
	switch runtime.GOOS {
	case "darwin": // macOS
		return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	case "linux":
		// Проверяем несколько возможных путей на Linux
		paths := []string{
			"/usr/bin/google-chrome-stable",
			"/usr/bin/google-chrome",
			"/opt/google/chrome/chrome",
		}
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
		return "/usr/bin/google-chrome-stable" // Fallback
	case "windows":
		return `C:\Program Files\Google\Chrome\Application\chrome.exe`
	default:
		return ""
	}
}

func generateSessionID(instanceID string) string {
	// Генерируем короткий случайный ID
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomID := fmt.Sprintf("%x", randomBytes)
	
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	
	if instanceID != "" {
		return fmt.Sprintf("session_%s_inst%s_%s", timestamp, instanceID, randomID)
	}
	return fmt.Sprintf("session_%s_%s", timestamp, randomID)
}

func uploadSingleFile(filePath, sessionID string) error {
	fileName := filepath.Base(filePath)
	
	// Создаем lftp скрипт для загрузки одного файла
	lftpScript := fmt.Sprintf(`open sftp://storage:storage123@kryuchenko.org
cd files/%s
put %s -o %s
quit
`, sessionID, filePath, fileName)

	// Создаем временный файл со скриптом
	tmpFile := fmt.Sprintf("/tmp/lftp_single_%d.txt", time.Now().UnixNano())
	err := os.WriteFile(tmpFile, []byte(lftpScript), 0644)
	if err != nil {
		return fmt.Errorf("failed to create lftp script: %v", err)
	}
	defer os.Remove(tmpFile)

	// Выполняем lftp команды
	cmd := exec.Command("lftp", "-f", tmpFile)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("lftp single upload output: %s", string(output))
		return fmt.Errorf("lftp single upload failed: %v", err)
	}

	log.Printf("Uploaded: %s", fileName)
	return nil
}

func uploadToRemote(localDir, sessionID string) error {
	// Используем lftp для SFTP загрузки (более надежный для автоматизации)
	files, err := filepath.Glob(filepath.Join(localDir, "*"))
	if err != nil {
		return fmt.Errorf("failed to list files: %v", err)
	}

	log.Printf("Uploading %d files to remote storage via SFTP...", len(files))

	// Создаем lftp скрипт (загружаем в папку files)
	lftpScript := fmt.Sprintf(`open sftp://storage:storage123@kryuchenko.org
cd files
mkdir %s
cd %s
`, sessionID, sessionID)

	for _, file := range files {
		fileName := filepath.Base(file)
		lftpScript += fmt.Sprintf("put %s -o %s\n", file, fileName)
	}
	lftpScript += "quit\n"

	// Создаем временный файл со скриптом
	tmpFile := "/tmp/lftp_script.txt"
	err = os.WriteFile(tmpFile, []byte(lftpScript), 0644)
	if err != nil {
		return fmt.Errorf("failed to create lftp script: %v", err)
	}
	defer os.Remove(tmpFile)

	// Выполняем lftp команды
	cmd := exec.Command("lftp", "-f", tmpFile)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("lftp output: %s", string(output))
		return fmt.Errorf("lftp upload failed: %v", err)
	}

	log.Printf("All artifacts uploaded to: storage@kryuchenko.org:~/files/%s/", sessionID)
	log.Printf("Upload output: %s", string(output))
	return nil
}

func createReport(artifacts []string, sessionDuration time.Duration) string {
	reportContent := fmt.Sprintf(`# Playwright Session Report

## Session Info
- Start Time: %s
- Duration: %s
- URL: https://x.la/cgs/1754888695/play

## Artifacts
`, time.Now().Add(-sessionDuration).Format("2006-01-02 15:04:05"), sessionDuration)

	for i, artifact := range artifacts {
		reportContent += fmt.Sprintf("- Screenshot %d: %s\n", i+1, filepath.Base(artifact))
	}

	reportContent += "\n## Session completed successfully\n"

	return reportContent
}

func main() {
	// Парсим аргументы командной строки
	instanceID := flag.String("instance", "", "Instance ID for unique session naming")
	flag.Parse()
	
	url := "https://x.la/cgs/1754888695/play"
	sessionDuration := 5 * time.Minute
	
	// Генерируем уникальный ID сессии
	sessionID := generateSessionID(*instanceID)
	log.Printf("Session ID: %s", sessionID)

	var err error

	// Монтируем SFTP как локальную файловую систему
	mountPoint := "./remote_artifacts"
	artifactsDir := filepath.Join(mountPoint, sessionID)
	
	// Проверяем наличие sshfs
	_, err = exec.LookPath("sshfs")
	canMount := err == nil
	
	if canMount {
		// Создаем точку монтирования
		err = os.MkdirAll(mountPoint, 0755)
		if err != nil {
			log.Fatalf("Could not create mount point: %v", err)
		}
		
		// Монтируем SFTP
		log.Println("Mounting SFTP filesystem...")
		cmd := exec.Command("sshfs", 
			"storage@kryuchenko.org:files", 
			mountPoint,
			"-o", "password_stdin",
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null")
		
		cmd.Stdin = strings.NewReader("storage123\n")
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("SSHFS mount failed: %v, output: %s", err, string(output))
			log.Println("Falling back to local directory...")
			artifactsDir = "./artifacts"
			canMount = false
		} else {
			log.Printf("SFTP mounted at %s", mountPoint)
			// Создаем папку для этой сессии
			err = os.MkdirAll(artifactsDir, 0755)
			if err != nil {
				log.Printf("Could not create session directory: %v", err)
				artifactsDir = "./artifacts"
				canMount = false
			}
		}
	} else {
		log.Println("Warning: sshfs not found, using local directory")
		log.Println("To install sshfs: brew install sshfs (macOS) or apt install sshfs (Linux)")
		artifactsDir = "./artifacts"
	}
	
	// Создаем локальную папку как fallback
	if !canMount {
		artifactsDir = "./artifacts"
		err = os.MkdirAll(artifactsDir, 0755)
		if err != nil {
			log.Fatalf("Could not create artifacts directory: %v", err)
		}
		
		// Проверяем наличие lftp для загрузки
		_, err = exec.LookPath("lftp")
		if err != nil {
			log.Fatalf("Neither sshfs nor lftp available - cannot save artifacts remotely")
		}
		log.Println("Will use lftp for uploading artifacts")
	}

	// Запускаем Playwright
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Could not start playwright: %v", err)
	}
	defer pw.Stop()

	// Получаем путь к Chrome для текущей ОС
	chromePath := getChromePath()
	if chromePath == "" {
		log.Fatalf("Could not find Google Chrome installation")
	}

	// Запускаем именно Google Chrome (не Chromium)
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless:       playwright.Bool(false), // Запускаем с GUI
		ExecutablePath: playwright.String(chromePath), // Используем системный Chrome
		Devtools:       playwright.Bool(false), // Без devtools
		Args: []string{
			"--password-store=basic",
			"--no-first-run",
			"--disable-features=AccountConsistency", 
			"--disable-signin-promo",
			"--start-maximized", // Запускаем на весь экран
			"--disable-web-security", // Отключаем ограничения безопасности
		},
	})
	if err != nil {
		log.Fatalf("Could not launch Chrome: %v", err)
	}
	log.Println("Chrome browser launched successfully")

	// Создаем новую страницу
	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("Could not create page: %v", err)
	}

	// Переходим по ссылке
	_, err = page.Goto(url)
	if err != nil {
		log.Fatalf("Could not goto URL: %v", err)
	}

	log.Printf("Successfully opened %s in Chrome via Playwright", url)
	log.Printf("Session will run for %v...", sessionDuration)

	// Ждем и кликаем по GDPR кнопке
	log.Println("Waiting for GDPR consent button...")
	gdprButton := page.Locator("#accept-button")
	err = gdprButton.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(30000), // 30 секунд ожидания
	})
	if err != nil {
		log.Printf("GDPR button not found within 30 seconds: %v", err)
	} else {
		err = gdprButton.Click()
		if err != nil {
			log.Printf("Could not click GDPR button: %v", err)
		} else {
			log.Println("GDPR consent accepted!")
		}
	}

	startTime := time.Now()
	var screenshots []string

	// Делаем скриншоты каждые 15 секунд
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	sessionTimer := time.NewTimer(sessionDuration)
	defer sessionTimer.Stop()

	screenshotCount := 0

	for {
		select {
		case <-ticker.C:
			screenshotCount++
			screenshotPath := filepath.Join(artifactsDir, fmt.Sprintf("screenshot_%d.png", screenshotCount))
			_, err := page.Screenshot(playwright.PageScreenshotOptions{
				Path: playwright.String(screenshotPath),
			})
			if err != nil {
				log.Printf("Could not take screenshot: %v", err)
			} else {
				screenshots = append(screenshots, screenshotPath)
				if canMount {
					log.Printf("Screenshot %d saved directly to remote: %s", screenshotCount, screenshotPath)
				} else {
					log.Printf("Screenshot %d saved locally: %s", screenshotCount, screenshotPath)
					// Сразу загружаем скриншот на удаленный диск через lftp
					go func(path string) {
						err := uploadSingleFile(path, sessionID)
						if err != nil {
							log.Printf("Failed to upload %s: %v", filepath.Base(path), err)
						}
					}(screenshotPath)
				}
			}

		case <-sessionTimer.C:
			log.Println("Session completed, generating report...")
			
			// Закрываем страницу и браузер
			page.Close()
			browser.Close()

			// Создаем отчет
			reportContent := createReport(screenshots, time.Since(startTime))
			reportPath := filepath.Join(artifactsDir, "session_report.md")
			err := os.WriteFile(reportPath, []byte(reportContent), 0644)
			if err != nil {
				log.Printf("Could not write report: %v", err)
			}

			// Размонтируем SFTP если использовали, иначе загружаем через lftp
			if canMount {
				log.Println("Unmounting SFTP filesystem...")
				cmd := exec.Command("umount", mountPoint)
				output, err := cmd.CombinedOutput()
				if err != nil {
					log.Printf("Warning: Could not unmount SFTP: %v, output: %s", err, string(output))
				} else {
					log.Println("SFTP unmounted successfully!")
				}
			} else {
				// Загружаем оставшиеся артефакты через lftp
				log.Println("Uploading remaining artifacts via lftp...")
				err := uploadToRemote(artifactsDir, sessionID)
				if err != nil {
					log.Printf("Warning: Could not upload artifacts: %v", err)
				} else {
					log.Println("All artifacts uploaded successfully!")
				}
			}

			log.Println("Session completed successfully!")
			return
		}
	}
}
