package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	respData := GetOffers()
	var ids []int
	for _, offer := range respData.Offers {
		if offer.Dph_total <= 1 {
			ids = append(ids, offer.Id)
		}
	}

	fmt.Print(ids)
}

type OffersResponse struct {
	Success bool `json:"success"`
	Offers  []struct {
		Id        int     `json:"id"`
		Dph_total float64 `json:"dph_total"`
	} `json:"offers"`
}

func GetOffers() OffersResponse {
	url := "https://console.vast.ai/api/v0/search/asks/"

	var body OffersResponse
	jsonData, err := json.Marshal(body)
	if err != nil {
		log.Fatalf("Ошибка маршалинга JSON: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Ошибка создания запроса: %v", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Ошибка чтения ответа: %v", err)
	}

	var respData OffersResponse
	err = json.Unmarshal(bodyBytes, &respData)
	if err != nil {
		log.Fatalf("Ошибка декодирования JSON: %v", err)
	}
	return respData
}

func StartInstance(id int) []byte {
	url := fmt.Sprintf("https://console.vast.ai/api/v0/asks/%v/", id)

	// Тело запроса
	payload := map[string]interface{}{
		"body": map[string]interface{}{
			"disk":         10,
			"env":          map[string]interface{}{},
			"target_state": "running",
			"vm":           false,
			"volume_info":  map[string]interface{}{},
		},
	}

	// Сериализация в JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Ошибка маршалинга JSON: %v", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Ошибка создания запроса: %v", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Ошибка чтения ответа: %v", err)
	}
	return bodyBytes
}
