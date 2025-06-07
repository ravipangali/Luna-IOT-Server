package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ControlRequest represents the request body for control operations
type ControlRequest struct {
	DeviceID *uint  `json:"device_id,omitempty"`
	IMEI     string `json:"imei,omitempty"`
}

// ControlResponse represents the response for control operations
type ControlResponse struct {
	Success    bool        `json:"success"`
	Message    string      `json:"message"`
	DeviceInfo interface{} `json:"device_info,omitempty"`
	Response   interface{} `json:"control_response,omitempty"`
	Error      string      `json:"error,omitempty"`
}

const (
	baseURL = "http://localhost:8080/api/v1"
)

func main() {
	fmt.Println("=== Luna IoT Server - Oil & Electricity Control Test ===")
	fmt.Println()

	// Test device IMEI (replace with actual device IMEI from your database)
	testIMEI := "123456789012345"

	// Test 1: Get active devices
	fmt.Println("1. Getting active devices...")
	getActiveDevices()
	fmt.Println()

	// Test 2: Cut oil and electricity
	fmt.Println("2. Testing Cut Oil and Electricity...")
	cutOilAndElectricity(testIMEI)
	fmt.Println()

	// Wait a bit
	time.Sleep(2 * time.Second)

	// Test 3: Connect oil and electricity
	fmt.Println("3. Testing Connect Oil and Electricity...")
	connectOilAndElectricity(testIMEI)
	fmt.Println()

	// Test 4: Get location
	fmt.Println("4. Testing Get Location...")
	getLocation(testIMEI)
	fmt.Println()

	// Test 5: Quick cut oil using device ID
	fmt.Println("5. Testing Quick Cut Oil (by device ID)...")
	quickCutOil(1) // Replace with actual device ID
	fmt.Println()

	// Test 6: Quick connect oil using IMEI
	fmt.Println("6. Testing Quick Connect Oil (by IMEI)...")
	quickConnectOilByIMEI(testIMEI)
	fmt.Println()

	fmt.Println("=== Test completed ===")
}

func getActiveDevices() {
	resp, err := http.Get(baseURL + "/control/active-devices")
	if err != nil {
		fmt.Printf("❌ Error getting active devices: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Error reading response: %v\n", err)
		return
	}

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(body))
}

func cutOilAndElectricity(imei string) {
	request := ControlRequest{
		IMEI: imei,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("❌ Error marshaling request: %v\n", err)
		return
	}

	resp, err := http.Post(baseURL+"/control/cut-oil", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Error sending cut oil request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Error reading response: %v\n", err)
		return
	}

	var response ControlResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("❌ Error parsing response: %v\n", err)
		fmt.Printf("Raw response: %s\n", string(body))
		return
	}

	fmt.Printf("Status: %d\n", resp.StatusCode)
	if response.Success {
		fmt.Printf("✅ %s\n", response.Message)
	} else {
		fmt.Printf("❌ %s\n", response.Message)
		if response.Error != "" {
			fmt.Printf("Error: %s\n", response.Error)
		}
	}
}

func connectOilAndElectricity(imei string) {
	request := ControlRequest{
		IMEI: imei,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("❌ Error marshaling request: %v\n", err)
		return
	}

	resp, err := http.Post(baseURL+"/control/connect-oil", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Error sending connect oil request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Error reading response: %v\n", err)
		return
	}

	var response ControlResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("❌ Error parsing response: %v\n", err)
		fmt.Printf("Raw response: %s\n", string(body))
		return
	}

	fmt.Printf("Status: %d\n", resp.StatusCode)
	if response.Success {
		fmt.Printf("✅ %s\n", response.Message)
	} else {
		fmt.Printf("❌ %s\n", response.Message)
		if response.Error != "" {
			fmt.Printf("Error: %s\n", response.Error)
		}
	}
}

func getLocation(imei string) {
	request := ControlRequest{
		IMEI: imei,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("❌ Error marshaling request: %v\n", err)
		return
	}

	resp, err := http.Post(baseURL+"/control/get-location", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Error sending get location request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Error reading response: %v\n", err)
		return
	}

	var response ControlResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("❌ Error parsing response: %v\n", err)
		fmt.Printf("Raw response: %s\n", string(body))
		return
	}

	fmt.Printf("Status: %d\n", resp.StatusCode)
	if response.Success {
		fmt.Printf("✅ %s\n", response.Message)
	} else {
		fmt.Printf("❌ %s\n", response.Message)
		if response.Error != "" {
			fmt.Printf("Error: %s\n", response.Error)
		}
	}
}

func quickCutOil(deviceID uint) {
	url := fmt.Sprintf("%s/control/quick-cut/%d", baseURL, deviceID)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		fmt.Printf("❌ Error sending quick cut oil request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Error reading response: %v\n", err)
		return
	}

	var response ControlResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("❌ Error parsing response: %v\n", err)
		fmt.Printf("Raw response: %s\n", string(body))
		return
	}

	fmt.Printf("Status: %d\n", resp.StatusCode)
	if response.Success {
		fmt.Printf("✅ %s\n", response.Message)
	} else {
		fmt.Printf("❌ %s\n", response.Message)
		if response.Error != "" {
			fmt.Printf("Error: %s\n", response.Error)
		}
	}
}

func quickConnectOilByIMEI(imei string) {
	url := fmt.Sprintf("%s/control/quick-connect-imei/%s", baseURL, imei)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		fmt.Printf("❌ Error sending quick connect oil request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Error reading response: %v\n", err)
		return
	}

	var response ControlResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("❌ Error parsing response: %v\n", err)
		fmt.Printf("Raw response: %s\n", string(body))
		return
	}

	fmt.Printf("Status: %d\n", resp.StatusCode)
	if response.Success {
		fmt.Printf("✅ %s\n", response.Message)
	} else {
		fmt.Printf("❌ %s\n", response.Message)
		if response.Error != "" {
			fmt.Printf("Error: %s\n", response.Error)
		}
	}
}
