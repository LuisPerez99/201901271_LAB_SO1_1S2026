package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pb "proyecto3/go-shared-pb/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// WarReport estructura para recibir reportes JSON
type WarReport struct {
	Country         string `json:"country"`
	WarplanesInAir  int32  `json:"warplanes_in_air"`
	WarshipsInWater int32  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

// GRPCRequest estructura para enviar al servidor gRPC
type GRPCRequest struct {
	Country         string
	WarplanesInAir  int32
	WarshipsInWater int32
	Timestamp       string
}

var (
	grpcServerAddr = os.Getenv("GRPC_SERVER_ADDR")
	httpPort       = os.Getenv("HTTP_PORT")
	mu             sync.Mutex
	requestCount   int64
	grpcClient     pb.WarReportServiceClient
)

func init() {
	if grpcServerAddr == "" {
		grpcServerAddr = "go-grpc-server:50051"
	}
	if httpPort == "" {
		httpPort = ":8080"
	}
}

func mapCountry(country string) (pb.Countries, error) {
	switch country {
	case "USA":
		return pb.Countries_usa, nil
	case "RUS":
		return pb.Countries_rus, nil
	case "CHN":
		return pb.Countries_chn, nil
	case "ESP":
		return pb.Countries_esp, nil
	case "GTM":
		return pb.Countries_gtm, nil
	default:
		return pb.Countries_countries_unknown, fmt.Errorf("invalid country %q", country)
	}
}

// handleReport maneja las solicitudes POST /report
func handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var report WarReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("[API-CLIENT] Received report: country=%s, planes=%d, ships=%d",
		report.Country, report.WarplanesInAir, report.WarshipsInWater)

	countryEnum, err := mapCountry(report.Country)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	grpcResp, err := grpcClient.SendReport(ctx, &pb.WarReportRequest{
		Country:         countryEnum,
		WarplanesInAir:  report.WarplanesInAir,
		WarshipsInWater: report.WarshipsInWater,
		Timestamp:       report.Timestamp,
	})
	if err != nil {
		log.Printf("[API-CLIENT] gRPC forward failed: %v", err)
		http.Error(w, "failed to forward report", http.StatusBadGateway)
		return
	}

	mu.Lock()
	requestCount++
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "accepted",
		"message":   grpcResp.GetMessage(),
		"report_id": grpcResp.GetReportId(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleMetrics devuelve métricas del servicio
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"requests_received": requestCount,
		"uptime":            time.Now().Unix(),
	})
}

// handleHealth health check
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"service":   "go-api-client",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func main() {
	log.Println("[GO-API-CLIENT] Starting service...")
	log.Printf("[GO-API-CLIENT] gRPC server: %s", grpcServerAddr)
	log.Printf("[GO-API-CLIENT] Listening on %s", httpPort)

	conn, err := grpc.Dial(grpcServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("[GO-API-CLIENT] Failed to connect gRPC server: %v", err)
	}
	defer conn.Close()
	grpcClient = pb.NewWarReportServiceClient(conn)

	// Setup HTTP routes
	http.HandleFunc("/report", handleReport)
	http.HandleFunc("/metrics", handleMetrics)
	http.HandleFunc("/health", handleHealth)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("[GO-API-CLIENT] Shutting down...")
		os.Exit(0)
	}()

	if err := http.ListenAndServe(httpPort, nil); err != nil {
		log.Fatalf("[GO-API-CLIENT] Server error: %v", err)
	}
}
