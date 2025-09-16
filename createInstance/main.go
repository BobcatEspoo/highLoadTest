package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	VASTAI_API_KEY = "1e19fbc2284ced113253ec19f519fc9aebbb2a36a917ee97b5c71376b2a2cf38"
	VASTAI_API_URL = "https://console.vast.ai/api/v0"
)

type VastClient struct {
	apiKey string
	client *http.Client
}

type Instance struct {
	ID           int     `json:"id"`
	Status       string  `json:"actual_status"`
	SSHHost      string  `json:"ssh_host"`
	SSHPort      int     `json:"ssh_port"`
	PublicIPAddr string  `json:"public_ipaddr"`
	DiskSpace    float64 `json:"disk_space"`
	GPUName      string  `json:"gpu_name"`
	NumGPUs      int     `json:"num_gpus"`
}

type CreateInstanceRequest struct {
	ClientID      string `json:"client_id"`
	Image         string `json:"image"`
	DiskSpace     int    `json:"disk"`
	OnStart       string `json:"onstart_cmd"`
	RunType       string `json:"runtype"`
	ImageLogin    string `json:"image_login"`
	PythonVersion string `json:"python_utf8"`
	CudaVersion   string `json:"cuda_version"`
	UseSSHKey     bool   `json:"use_ssh_key"`
}

type Offer struct {
	ID           int     `json:"id"`
	GPUName      string  `json:"gpu_name"`
	NumGPUs      int     `json:"num_gpus"`
	DiskSpace    float64 `json:"disk_space"`
	DPHTotal     float64 `json:"dph_total"`
	CudaMaxGood  float64 `json:"cuda_max_good"`
	Rentable     bool    `json:"rentable"`
	MinBid       float64 `json:"min_bid"`
	Verification string  `json:"verification"`
}

func NewVastClient(apiKey string) *VastClient {
	return &VastClient{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (v *VastClient) makeRequest(method, endpoint string, body interface{}) ([]byte, error) {
	url := VASTAI_API_URL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	// Используем те же заголовки что и UI
	req.Header.Set("Authorization", "Bearer "+v.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.9,ru-RU;q=0.8,ru;q=0.7,en-US;q=0.6")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Origin", "https://cloud.vast.ai")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="140", "Not=A?Brand";v="24", "Google Chrome";v="140"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36")

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (v *VastClient) SearchOffers(verifiedOnly bool) ([]Offer, error) {
	searchQuery := map[string]interface{}{
		"rentable":             map[string]bool{"eq": true},
		"disk_space":           map[string]float64{"gte": 30.0},
		"gpu_display_active":   map[string]bool{"eq": true},
	}
	
	// Добавляем фильтр верификации если нужно
	if verifiedOnly {
		searchQuery["verification"] = map[string]string{"eq": "verified"}
	}

	queryJSON, _ := json.Marshal(searchQuery)
	endpoint := fmt.Sprintf("/bundles?q=%s&order=dph_total&type=on-demand&limit=10", url.QueryEscape(string(queryJSON)))

	data, err := v.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Offers []Offer `json:"offers"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result.Offers, nil
}

var sshKeySetOnce sync.Once
var globalSSHKeyError error

func (v *VastClient) CreateInstance(offer Offer) (*Instance, error) {
	// Устанавливаем SSH ключ только один раз глобально
	sshKeySetOnce.Do(func() {
		sshKey, err := getOrCreateSSHKey()
		if err != nil {
			globalSSHKeyError = fmt.Errorf("failed to get SSH key: %v", err)
			return
		}

		// Skip API attempt, use CLI directly
		fmt.Println("Setting SSH key via CLI...")
		cmd := exec.Command("vastai", "create", "ssh-key", sshKey)
		output, cmdErr := cmd.CombinedOutput()
		if cmdErr != nil {
			globalSSHKeyError = fmt.Errorf("failed to add SSH key via CLI: %v\nOutput: %s", cmdErr, string(output))
			return
		}
		fmt.Printf("SSH key added via CLI: %s\n", string(output))
		fmt.Println("SSH key configured successfully")
	})

	if globalSSHKeyError != nil {
		return nil, globalSSHKeyError
	}

	fmt.Println("Creating instance via API...")
	
	// Создаем через API с правильной структурой как в UI
	portalConfig := "localhost:1111:11111:/:Instance Portal|localhost:6100:16100:/:Selkies Low Latency Desktop|localhost:6200:16200:/guacamole:Apache Guacamole Desktop (VNC)|localhost:8080:8080:/:Jupyter|localhost:8080:8080:/terminals/1:Jupyter Terminal|localhost:8384:18384:/:Syncthing"
	
	endpoint := "/asks/" + fmt.Sprintf("%d", offer.ID) + "/"
	
	// Используем точную структуру из curl запроса
	data := map[string]interface{}{
		"template_id":      246282,
		"template_hash_id": "5ba61f6f48c1ffe49ca7163d4e3a6460",
		"client_id":        "me",
		"image":            "vastai/linux-desktop:@vastai-automatic-tag",
		"env": map[string]string{
			"-p 1111:1111":      "1",
			"-p 6100:6100":      "1",
			"-p 73478:73478":    "1",
			"-p 8384:8384":      "1",
			"-p 72299:72299":    "1",
			"-p 6200:6200":      "1",
			"-p 5900:5900":      "1",
			"OPEN_BUTTON_TOKEN": "1",
			"JUPYTER_DIR":       "/",
			"DATA_DIRECTORY":    "/workspace/",
			"PORTAL_CONFIG":     portalConfig,
			"OPEN_BUTTON_PORT":  "1111",
			"SELKIES_ENCODER":   "x264enc",
		},
		"args_str":              "",
		"onstart":               "entrypoint.sh",
		"runtype":               "jupyter_direc ssh_direc ssh_proxy",
		"image_login":           nil,
		"use_jupyter_lab":       false,
		"jupyter_dir":           nil,
		"python_utf8":           nil,
		"lang_utf8":             nil,
		"disk":                  32,
		"last_known_min_bid":    offer.MinBid,
		"min_duration":          259200,
	}
	
	respBody, err := v.makeRequest("PUT", endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance via API: %v", err)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %v", err)
	}
	
	contractID, ok := result["new_contract"].(float64)
	if !ok {
		return nil, fmt.Errorf("failed to get contract ID from API response: %v", result)
	}

	fmt.Printf("Instance created with ID: %d\n", int(contractID))
	return &Instance{ID: int(contractID)}, nil
}

func (v *VastClient) SetSSHKey(publicKey string) error {
	endpoint := "/users/current/"

	updateData := map[string]string{
		"ssh_key": publicKey,
	}

	_, err := v.makeRequest("PUT", endpoint, updateData)
	return err
}

func (v *VastClient) GetInstance(instanceID int) (*Instance, error) {
	cmd := exec.Command("vastai", "show", "instance", fmt.Sprintf("%d", instanceID))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get instance info: %v\nOutput: %s", err, string(output))
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected output format: %s", string(output))
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 11 {
		return nil, fmt.Errorf("not enough fields in output: %s", lines[1])
	}

	instance := &Instance{
		ID:     instanceID,
		Status: fields[2],
	}

	// Если инстанс готов, получаем правильные SSH данные через ssh-url
	if fields[2] == "running" {
		sshCmd := exec.Command("vastai", "ssh-url", fmt.Sprintf("%d", instanceID))
		sshOutput, err := sshCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to get SSH URL: %v\nOutput: %s", err, string(sshOutput))
		}
		
		// Парсим ssh://root@host:port
		sshURL := strings.TrimSpace(string(sshOutput))
		if strings.HasPrefix(sshURL, "ssh://root@") {
			hostPort := strings.TrimPrefix(sshURL, "ssh://root@")
			parts := strings.Split(hostPort, ":")
			if len(parts) == 2 {
				instance.SSHHost = parts[0]
				port, err := strconv.Atoi(parts[1])
				if err != nil {
					return nil, fmt.Errorf("invalid SSH port: %s", parts[1])
				}
				instance.SSHPort = port
			}
		}
	}

	return instance, nil
}

func (v *VastClient) WaitForInstance(instanceID int, maxMinutes int) (*Instance, error) {
	maxAttempts := maxMinutes * 12 // 12 attempts per minute (every 5 seconds)
	for i := 0; i < maxAttempts; i++ {
		instance, err := v.GetInstance(instanceID)
		if err != nil {
			return nil, err
		}

		if instance.Status == "running" && instance.SSHHost != "" {
			return instance, nil
		}

		fmt.Printf("Instance status: %s (attempt %d/%d)\n", instance.Status, i+1, maxAttempts)
		time.Sleep(5 * time.Second)
	}

	return nil, fmt.Errorf("instance did not become ready in %d minutes", maxMinutes)
}

func getOrCreateSSHKey() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	sshDir := fmt.Sprintf("%s/.ssh", homeDir)
	keyPath := fmt.Sprintf("%s/vastai_rsa", sshDir)
	pubKeyPath := keyPath + ".pub"

	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		fmt.Println("Creating new SSH key pair...")

		os.MkdirAll(sshDir, 0700)

		cmd := exec.Command("ssh-keygen", "-t", "rsa", "-b", "4096", "-f", keyPath, "-N", "", "-C", "vastai-automation")
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to generate SSH key: %v", err)
		}

		fmt.Printf("SSH key pair created at %s\n", keyPath)
	}

	pubKey, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", err
	}

	return string(pubKey), nil
}

func connectSSH(instance *Instance) error {
	homeDir, _ := os.UserHomeDir()
	keyPath := fmt.Sprintf("%s/.ssh/vastai_rsa", homeDir)
	cmdStr := fmt.Sprintf("git clone https://github.com/kryuchenko/highLoadTest.git; cd highLoadTest; bash start.sh %v", instance.ID)
	sshTarget := fmt.Sprintf("root@%s", instance.SSHHost)
	fmt.Printf("\nConnecting to instance via SSH...\n")

	cmd := exec.Command("ssh",
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", strconv.Itoa(instance.SSHPort),
		sshTarget, cmdStr)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func startTestsOnInstances(instances []*Instance) error {
	fmt.Printf("\n=== STARTING TESTS ON %d INSTANCES ===\n", len(instances))
	fmt.Printf("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	
	// Выводим детали всех инстансов
	fmt.Printf("\nInstance details:\n")
	for i, inst := range instances {
		fmt.Printf("  [%d] ID: %d, SSH: %s:%d\n", i+1, inst.ID, inst.SSHHost, inst.SSHPort)
	}
	fmt.Printf("\n")
	
	var wg sync.WaitGroup
	var successCount, failCount int
	var mu sync.Mutex
	
	for i, instance := range instances {
		wg.Add(1)
		go func(idx int, inst *Instance) {
			defer wg.Done()
			
			fmt.Printf("[Instance %d/%d] [%s] Starting test on ID %d (%s:%d)...\n", 
				idx+1, len(instances), time.Now().Format("15:04:05"), inst.ID, inst.SSHHost, inst.SSHPort)
			
			// Первый этап - проверка подключения
			fmt.Printf("[Instance %d] [%s] Testing SSH connection...\n", inst.ID, time.Now().Format("15:04:05"))
			testCmd := exec.Command("ssh", 
				"-o", "StrictHostKeyChecking=no",
				"-o", "ConnectTimeout=10",
				"-p", strconv.Itoa(inst.SSHPort),
				fmt.Sprintf("root@%s", inst.SSHHost),
				"echo 'SSH connection successful'")
			
			output, err := testCmd.CombinedOutput()
			if err != nil {
				fmt.Printf("[Instance %d] [%s] SSH connection failed: %v, output: %s\n", 
					inst.ID, time.Now().Format("15:04:05"), err, string(output))
				mu.Lock()
				failCount++
				mu.Unlock()
				return
			}
			fmt.Printf("[Instance %d] [%s] SSH connection OK: %s\n", 
				inst.ID, time.Now().Format("15:04:05"), strings.TrimSpace(string(output)))
			
			// Второй этап - скачивание файлов
			fmt.Printf("[Instance %d] [%s] Downloading test files...\n", inst.ID, time.Now().Format("15:04:05"))
			downloadCmd := exec.Command("ssh", 
				"-o", "StrictHostKeyChecking=no",
				"-o", "ConnectTimeout=30",
				"-p", strconv.Itoa(inst.SSHPort),
				fmt.Sprintf("root@%s", inst.SSHHost),
				`wget -O highLoadTest https://github.com/kryuchenko/highLoadTest/raw/add-price-limit/highLoadTest && 
				 wget -O start.sh https://github.com/kryuchenko/highLoadTest/raw/add-price-limit/start.sh && 
				 chmod +x start.sh && 
				 ls -la highLoadTest start.sh`)
			
			output, err = downloadCmd.CombinedOutput()
			if err != nil {
				fmt.Printf("[Instance %d] [%s] Download failed: %v, output: %s\n", 
					inst.ID, time.Now().Format("15:04:05"), err, string(output))
				mu.Lock()
				failCount++
				mu.Unlock()
				return
			}
			fmt.Printf("[Instance %d] [%s] Download successful:\n%s\n", 
				inst.ID, time.Now().Format("15:04:05"), string(output))
			
			// Третий этап - запуск теста
			fmt.Printf("[Instance %d] [%s] Starting test execution...\n", inst.ID, time.Now().Format("15:04:05"))
			runCmd := exec.Command("ssh", 
				"-o", "StrictHostKeyChecking=no",
				"-o", "ConnectTimeout=30",
				"-p", strconv.Itoa(inst.SSHPort),
				fmt.Sprintf("root@%s", inst.SSHHost),
				fmt.Sprintf(`nohup ./start.sh %d > test_output.log 2>&1 & echo "Test started with PID: $!"`, inst.ID))
			
			output, err = runCmd.CombinedOutput()
			if err != nil {
				fmt.Printf("[Instance %d] [%s] Test start failed: %v, output: %s\n", 
					inst.ID, time.Now().Format("15:04:05"), err, string(output))
				mu.Lock()
				failCount++
				mu.Unlock()
				return
			}
			
			fmt.Printf("[Instance %d] [%s] Test started successfully! %s\n", 
				inst.ID, time.Now().Format("15:04:05"), strings.TrimSpace(string(output)))
			
			// Проверяем что процесс действительно запустился
			time.Sleep(2 * time.Second)
			checkCmd := exec.Command("ssh", 
				"-o", "StrictHostKeyChecking=no",
				"-o", "ConnectTimeout=10",
				"-p", strconv.Itoa(inst.SSHPort),
				fmt.Sprintf("root@%s", inst.SSHHost),
				"ps aux | grep highLoadTest | grep -v grep")
			
			output, err = checkCmd.CombinedOutput()
			if err == nil && len(output) > 0 {
				fmt.Printf("[Instance %d] [%s] Process confirmed running: %s\n", 
					inst.ID, time.Now().Format("15:04:05"), strings.TrimSpace(string(output)))
			} else {
				fmt.Printf("[Instance %d] [%s] Warning: Process not found in ps output\n", 
					inst.ID, time.Now().Format("15:04:05"))
			}
			
			mu.Lock()
			successCount++
			mu.Unlock()
		}(i, instance)
	}
	
	fmt.Printf("\nWaiting for all test deployments to complete...\n")
	wg.Wait()
	
	fmt.Printf("\n=== TEST DEPLOYMENT SUMMARY ===\n")
	fmt.Printf("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("Total instances: %d\n", len(instances))
	fmt.Printf("Successfully started: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failCount)
	fmt.Printf("Success rate: %.1f%%\n", float64(successCount)/float64(len(instances))*100)
	fmt.Printf("\nMonitor results at: storage@kryuchenko.org:~/files/\n")
	fmt.Printf("Expected session IDs format: session_YYYY-MM-DD_HH-MM-SS_instINSTANCE_ID_RANDOM\n")
	
	if successCount > 0 {
		fmt.Printf("\n=== NEXT STEPS ===\n")
		fmt.Printf("1. Wait 1-2 minutes for Chrome to start and GDPR consent\n")
		fmt.Printf("2. Check remote storage for new session folders\n")
		fmt.Printf("3. Screenshots will be saved every 15 seconds\n")
		fmt.Printf("4. Tests will run for 5 minutes each\n")
	}
	
	return nil
}

func main() {
	count := flag.Int("count", 1, "how many instances needed")
	waitMinutes := flag.Int("wait", 5, "minutes to wait for instances to be ready")
	maxPrice := flag.Float64("max-price", 0.50, "maximum price per hour in USD")
	startTests := flag.Bool("start-tests", false, "start tests on created instances after waiting")
	verifiedOnly := flag.Bool("verified", false, "use only verified instances")
	flag.Parse()
	client := NewVastClient(VASTAI_API_KEY)

	if *verifiedOnly {
		fmt.Println("Searching for VERIFIED GPU instances...")
	} else {
		fmt.Println("Searching for ALL available GPU instances...")
	}
	offers, err := client.SearchOffers(*verifiedOnly)
	if err != nil {
		log.Fatalf("Failed to search offers: %v", err)
	}

	if len(offers) == 0 {
		log.Fatal("No suitable offers found")
	}

	fmt.Printf("Found %d offers. Filtering by max price $%.2f/hour...\n", len(offers), *maxPrice)

	// Filter by price first
	var filteredOffers []Offer
	for _, offer := range offers {
		if offer.DPHTotal <= *maxPrice {
			filteredOffers = append(filteredOffers, offer)
		}
	}
	
	if len(filteredOffers) == 0 {
		log.Fatalf("No offers found under $%.2f/hour", *maxPrice)
	}
	
	fmt.Printf("Found %d offers under $%.2f/hour. Selecting the cheapest ones...\n", len(filteredOffers), *maxPrice)

	sort.Slice(filteredOffers, func(i, j int) bool {
		return filteredOffers[i].DPHTotal < filteredOffers[j].DPHTotal
	})
	
	// Remove expensive GPUs  
	for i := len(filteredOffers) - 1; i >= 0; i-- {
		if strings.Contains(filteredOffers[i].GPUName, "3090") || strings.Contains(filteredOffers[i].GPUName, "4090") {
			filteredOffers = append(filteredOffers[:i], filteredOffers[i+1:]...)
		}
	}
	
	offers = filteredOffers
	offersSlice := offers[:*count]
	var wg sync.WaitGroup
	var mu sync.Mutex
	var createdInstances []*Instance

	// Параллельное создание экземпляров с ограничением одновременных запросов
	fmt.Printf("Creating %d instances with rate limiting...\n", len(offersSlice))

	// Канал для ограничения количества одновременных запросов (учитываем rate limit)
	maxConcurrent := 1 // Максимум 1 запрос за раз для избежания 429 ошибок
	semaphore := make(chan struct{}, maxConcurrent)

	for i, offer := range offersSlice {
		wg.Add(1)
		go func(offerIndex int, offer Offer) {
			defer wg.Done()

			// Захватываем слот в семафоре
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fmt.Printf("\n[Instance %d] [%s] Selected offer:\n", offerIndex+1, time.Now().Format("15:04:05"))
			fmt.Printf("  GPU: %s x%d\n", offer.GPUName, offer.NumGPUs)
			fmt.Printf("  Disk: %.1f GB\n", offer.DiskSpace)
			fmt.Printf("  Price: $%.4f/hour\n", offer.DPHTotal)
			fmt.Printf("  Offer ID: %d\n", offer.ID)
			fmt.Printf("  Verification: %s\n", offer.Verification)
			fmt.Printf("  Min Bid: $%.4f/hour\n\n", offer.MinBid)

			fmt.Printf("[Instance %d] [%s] Creating instance...\n", offerIndex+1, time.Now().Format("15:04:05"))

			// Фиксированная задержка для соблюдения rate limit (4 req/sec = 250ms между запросами)
			time.Sleep(250 * time.Millisecond)

			instance, err := client.CreateInstance(offer)
			if err != nil {
				fmt.Printf("[Instance %d] [%s] Failed to create instance: %v\n", offerIndex+1, time.Now().Format("15:04:05"), err)
				return
			}

			fmt.Printf("[Instance %d] [%s] Instance created successfully!\n", offerIndex+1, time.Now().Format("15:04:05"))
			fmt.Printf("  Instance ID: %d\n", instance.ID)
			fmt.Printf("  Offer ID: %d\n", offer.ID)
			fmt.Printf("  Expected session format: session_*_inst%d_*\n", instance.ID)

			mu.Lock()
			createdInstances = append(createdInstances, instance)
			mu.Unlock()
		}(i, offer)
	}

	// Ждем завершения создания всех экземпляров
	wg.Wait()

	fmt.Printf("\n=== ALL INSTANCES CREATED ===\n")
	fmt.Printf("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("Created %d instances, waiting %d minutes for them to be ready...\n", len(createdInstances), *waitMinutes)
	fmt.Printf("Expected initialization time: 2-10 minutes depending on instance type\n")
	
	if len(createdInstances) > 0 {
		fmt.Printf("\nCreated instance IDs:\n")
		for i, inst := range createdInstances {
			fmt.Printf("  [%d] Instance ID: %d\n", i+1, inst.ID)
		}
	}

	fmt.Printf("\nWaiting %d minutes for instance initialization...\n", *waitMinutes)
	for i := 1; i <= *waitMinutes; i++ {
		fmt.Printf("\n  [%s] Minute %d/%d - Checking instance statuses:\n", time.Now().Format("15:04:05"), i, *waitMinutes)
		
		// Проверяем статус каждого инстанса каждую минуту
		readyCount := 0
		for _, instance := range createdInstances {
			statusInstance, err := client.GetInstance(instance.ID)
			if err != nil {
				fmt.Printf("    Instance %d: ERROR checking status\n", instance.ID)
				continue
			}
			
			statusIcon := "⏳"
			if statusInstance.Status == "running" {
				statusIcon = "✅"
				readyCount++
			} else if statusInstance.Status == "loading" {
				statusIcon = "🔄"
			} else if statusInstance.Status == "created" {
				statusIcon = "🆕"
			} else if statusInstance.Status == "error" {
				statusIcon = "❌"
			}
			
			sshStatus := ""
			if statusInstance.Status == "running" && statusInstance.SSHHost != "" {
				// Быстрая проверка SSH доступности
				cmd := exec.Command("nc", "-z", "-w", "1", statusInstance.SSHHost, strconv.Itoa(statusInstance.SSHPort))
				if err := cmd.Run(); err == nil {
					sshStatus = " (SSH ready)"
				} else {
					sshStatus = " (SSH not ready)"
				}
			}
			
			fmt.Printf("    %s Instance %d: %s%s\n", statusIcon, instance.ID, statusInstance.Status, sshStatus)
		}
		
		fmt.Printf("  Ready: %d/%d instances\n", readyCount, len(createdInstances))
		
		// Если все готовы, можем прервать ожидание
		if readyCount == len(createdInstances) && i < *waitMinutes {
			fmt.Printf("\n  🎉 All instances ready! Proceeding early...\n")
			break
		}
		
		if i < *waitMinutes {
			time.Sleep(1 * time.Minute)
		}
	}

	// Собираем готовые экземпляры
	fmt.Printf("\n=== CHECKING READY INSTANCES ===\n")
	fmt.Printf("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	var readyInstances []*Instance

	for i, instance := range createdInstances {
		fmt.Printf("[%d/%d] [%s] Checking instance %d... ", i+1, len(createdInstances), time.Now().Format("15:04:05"), instance.ID)
		readyInstance, err := client.GetInstance(instance.ID)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}

		if readyInstance.Status == "running" && readyInstance.SSHHost != "" {
			fmt.Printf("READY (Host: %s, Port: %d)\n", readyInstance.SSHHost, readyInstance.SSHPort)
			readyInstances = append(readyInstances, readyInstance)
		} else {
			fmt.Printf("NOT READY (Status: %s)\n", readyInstance.Status)
		}
	}

	fmt.Printf("\n=== POOL READY ===\n")
	fmt.Printf("Ready instances: %d/%d\n", len(readyInstances), len(createdInstances))

	// Подключение к готовым экземплярам
	if len(readyInstances) > 0 {
		// Запускаем тесты если флаг установлен
		if *startTests {
			err := startTestsOnInstances(readyInstances)
			if err != nil {
				fmt.Printf("Error starting tests: %v\n", err)
			}
		}
	} else {
		fmt.Printf("\nNo ready instances to connect to.\n")
	}
}
