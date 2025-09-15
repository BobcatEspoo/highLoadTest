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
	ID          int     `json:"id"`
	GPUName     string  `json:"gpu_name"`
	NumGPUs     int     `json:"num_gpus"`
	DiskSpace   float64 `json:"disk_space"`
	DPHTotal    float64 `json:"dph_total"`
	CudaMaxGood float64 `json:"cuda_max_good"`
	Rentable    bool    `json:"rentable"`
	MinBid      float64 `json:"min_bid"`
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

	req.Header.Set("Authorization", "Bearer "+v.apiKey)
	req.Header.Set("Content-Type", "application/json")

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

func (v *VastClient) SearchOffers() ([]Offer, error) {
	searchQuery := map[string]interface{}{
		"rentable":             map[string]bool{"eq": true},
		"disk_space":           map[string]float64{"gte": 30.0},
		"gpu_display_active":   map[string]bool{"eq": true},
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

func (v *VastClient) CreateInstance(offerID int) (*Instance, error) {
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

	fmt.Println("Creating instance via CLI...")
	envParams := "-p 1111:1111 -p 6100:6100 -p 73478:73478 -p 8384:8384 -p 72299:72299 -p 6200:6200 -p 5900:5900 -e OPEN_BUTTON_TOKEN=1 -e JUPYTER_DIR=/ -e DATA_DIRECTORY=/workspace/ -e PORTAL_CONFIG=localhost:1111:11111:/:Instance Portal|localhost:6100:16100:/:Selkies Low Latency Desktop|localhost:6200:16200:/guacamole:Apache Guacamole Desktop (VNC)|localhost:8080:8080:/:Jupyter|localhost:8080:8080:/terminals/1:Jupyter Terminal|localhost:8384:18384:/:Syncthing -e OPEN_BUTTON_PORT=1111 -e SELKIES_ENCODER=x264enc"
	
	cmd := exec.Command("vastai", "create", "instance", fmt.Sprintf("%d", offerID),
		"--image", "vastai/linux-desktop:@vastai-automatic-tag",
		"--env", envParams,
		"--onstart-cmd", "env | grep _ >> /etc/environment; echo \"$PORTAL_CONFIG\" | tr '|' '\\n' > /etc/portal.yaml; echo 'Portal config written to /etc/portal.yaml'; echo 'Desktop environment starting...'",
		"--disk", "32",
		"--jupyter",
		"--ssh",
		"--direct")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create instance via CLI: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	fmt.Printf("Instance creation output: %s\n", outputStr)

	var contractID int
	// Try parsing both success and failure cases since contract ID is created in both
	if _, err := fmt.Sscanf(outputStr, "Started. {'success': True, 'new_contract': %d}", &contractID); err != nil {
		if _, err2 := fmt.Sscanf(outputStr, "Started. {'success': False, 'new_contract': %d}", &contractID); err2 != nil {
			return nil, fmt.Errorf("failed to parse contract ID from output: %s", outputStr)
		}
	}

	fmt.Printf("Instance created with ID: %d\n", contractID)

	return &Instance{ID: contractID}, nil
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

	if fields[2] == "running" && len(fields) >= 11 {
		instance.SSHHost = fields[9]
		port, _ := strconv.Atoi(fields[10])
		instance.SSHPort = port
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
	cmdStr := fmt.Sprintf("git clone https://github.com/BobcatEspoo/highLoadTest.git; cd highLoadTest; bash start.sh %v", instance.ID)
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

func main() {
	count := flag.Int("count", 1, "how many instances needed")
	waitMinutes := flag.Int("wait", 5, "minutes to wait for instances to be ready")
	maxPrice := flag.Float64("max-price", 0.50, "maximum price per hour in USD")
	flag.Parse()
	client := NewVastClient(VASTAI_API_KEY)

	fmt.Println("Searching for available GPU instances...")
	offers, err := client.SearchOffers()
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

	// Канал для ограничения количества одновременных запросов
	maxConcurrent := 5 // Максимум 5 одновременных запросов
	semaphore := make(chan struct{}, maxConcurrent)

	for i, offer := range offersSlice {
		wg.Add(1)
		go func(offerIndex int, offer Offer) {
			defer wg.Done()

			// Захватываем слот в семафоре
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fmt.Printf("\n[Instance %d] Selected offer:\n", offerIndex+1)
			fmt.Printf("  GPU: %s x%d\n", offer.GPUName, offer.NumGPUs)
			fmt.Printf("  Disk: %.1f GB\n", offer.DiskSpace)
			fmt.Printf("  Price: $%.4f/hour\n", offer.DPHTotal)
			fmt.Printf("  Offer ID: %d\n\n", offer.ID)

			fmt.Printf("[Instance %d] Creating instance...\n", offerIndex+1)

			// Добавляем задержку между запросами
			time.Sleep(time.Duration(offerIndex) * 2 * time.Second)

			instance, err := client.CreateInstance(offer.ID)
			if err != nil {
				fmt.Printf("[Instance %d] Failed to create instance: %v\n", offerIndex+1, err)
				return
			}

			fmt.Printf("\n[Instance %d] Instance created with ID: %d\n", offerIndex+1, instance.ID)

			mu.Lock()
			createdInstances = append(createdInstances, instance)
			mu.Unlock()
		}(i, offer)
	}

	// Ждем завершения создания всех экземпляров
	wg.Wait()

	fmt.Printf("\n=== ALL INSTANCES CREATED ===\n")
	fmt.Printf("Created %d instances, waiting %d minutes for them to be ready...\n", len(createdInstances), *waitMinutes)

	// Ждем глобальный таймаут
	time.Sleep(time.Duration(*waitMinutes) * time.Minute)

	// Собираем готовые экземпляры
	fmt.Printf("\n=== CHECKING READY INSTANCES ===\n")
	var readyInstances []*Instance

	for _, instance := range createdInstances {
		fmt.Printf("Checking instance %d... ", instance.ID)
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
		fmt.Printf("\nConnecting to ready instances...\n")
		for i, instance := range readyInstances {
			fmt.Printf("\n[Instance %d/%d] Connecting to ID %d...\n", i+1, len(readyInstances), instance.ID)
			if err := connectSSH(instance); err != nil {
				fmt.Printf("SSH connection to instance %d closed or failed: %v\n", instance.ID, err)
			}
		}
	} else {
		fmt.Printf("\nNo ready instances to connect to.\n")
	}
}
