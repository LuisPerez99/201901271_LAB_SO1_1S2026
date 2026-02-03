package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type HealthResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	VM        string `json:"vm"`
	Carnet    string `json:"carnet"`
}

type ApiResponse struct {
	ApiName    string `json:"apiname"`
	Message    string `json:"message"`
	Connection bool   `json:"connection"`
	Carnet     string `json:"carnet"`
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	fmt.Fprintln(w, "API1 - 201901271")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := HealthResponse{
		Status:    "UP",
		Message:   "API1 is Ready",
		Timestamp: time.Now().Format(time.RFC3339),
		VM:        "1",
		Carnet:    "201901271",
	}

	json.NewEncoder(w).Encode(response)
}

func callApi2Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp, err := http.Get("http://" + os.Getenv("HOST_API2") + ":" + os.Getenv("PORT_API2") + "/health")
	if err != nil || resp.StatusCode != http.StatusOK || resp.Status != "UP" {
		writeApiResponse(w, "API2", "ERROR: The API2 located on the VM2 is not working", false)
		return
	}

	writeApiResponse(w, "API2", "The API2 located on the VM2 is working", true)
}

func callApi3Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp, err := http.Get("http://" + os.Getenv("HOST_API3") + ":" + os.Getenv("PORT_API3") + "/health")
	if err != nil || resp.StatusCode != http.StatusOK || resp.Status != "UP" {
		writeApiResponse(w, "API3", "ERROR: The API3 located on the VM3 is not working", false)
		return
	}

	writeApiResponse(w, "API3", "The API3 located on the VM3 is working", true)
}

func writeApiResponse(w http.ResponseWriter, apiName, message string, connection bool) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := ApiResponse{
		ApiName:    apiName,
		Message:    message,
		Connection: connection,
		Carnet:     "201901271",
	}

	json.NewEncoder(w).Encode(response)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	host_api1 := os.Getenv("HOST_API1")
	port_api1 := os.Getenv("PORT_API1")

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api1/201901271/call-api2", callApi2Handler)
	http.HandleFunc("/api1/201901271/call-api3", callApi3Handler)

	fmt.Printf("API 1 is running on http://%s:%s\n", host_api1, port_api1)
	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", host_api1, port_api1), nil); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
