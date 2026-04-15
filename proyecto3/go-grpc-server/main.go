package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	pb "proyecto3/go-shared-pb/pb"

	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	grpcPort      = os.Getenv("GRPC_PORT")
	rabbitmqAddr  = os.Getenv("RABBITMQ_ADDR")
	rabbitmqVhost = os.Getenv("RABBITMQ_VHOST")
	rabbitmqUser  = os.Getenv("RABBITMQ_USER")
	rabbitmqPass  = os.Getenv("RABBITMQ_PASS")

	// RabbitMQ connection
	conn *amqp.Connection
	ch   *amqp.Channel
	mu   sync.Mutex
)

type warReportServer struct {
	pb.UnimplementedWarReportServiceServer
}

func (s *warReportServer) SendReport(ctx context.Context, req *pb.WarReportRequest) (*pb.WarReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	countryName := req.GetCountry().String()
	if countryName == "countries_unknown" {
		return nil, status.Error(codes.InvalidArgument, "invalid country")
	}

	countryRouting := countryName
	payload := map[string]interface{}{
		"country":           countryName,
		"warplanes_in_air":  req.GetWarplanesInAir(),
		"warships_in_water": req.GetWarshipsInWater(),
		"timestamp":         req.GetTimestamp(),
		"received_at":       time.Now().UTC().Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal error: %v", err)
	}

	if err := publishToRabbitMQ(countryRouting, body); err != nil {
		return nil, status.Errorf(codes.Internal, "rabbitmq publish error: %v", err)
	}

	return &pb.WarReportResponse{
		Status:   "accepted",
		Message:  "report forwarded to RabbitMQ",
		ReportId: fmt.Sprintf("%d", time.Now().UnixNano()),
	}, nil
}

func init() {
	if grpcPort == "" {
		grpcPort = ":50051"
	}
	if rabbitmqAddr == "" {
		rabbitmqAddr = "rabbitmq:5672"
	}
	if rabbitmqVhost == "" {
		rabbitmqVhost = "/warreports"
	}
	if rabbitmqUser == "" {
		rabbitmqUser = "guest"
	}
	if rabbitmqPass == "" {
		rabbitmqPass = "guest"
	}
}

// initRabbitMQ inicializa la conexión a RabbitMQ
func initRabbitMQ() error {
	dsn := fmt.Sprintf("amqp://%s:%s@%s%s", rabbitmqUser, rabbitmqPass, rabbitmqAddr, rabbitmqVhost)

	var err error
	conn, err = amqp.Dial(dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err = conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to create channel: %w", err)
	}

	// Declare exchange
	err = ch.ExchangeDeclare(
		"warreports", // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queues for each country
	countries := []string{"usa", "rus", "chn", "esp", "gtm"}
	for _, country := range countries {
		queueName := fmt.Sprintf("reports.%s", country)
		_, err = ch.QueueDeclare(
			queueName, // name
			true,      // durable
			false,     // delete when unused
			false,     // exclusive
			false,     // no-wait
			nil,       // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
		}

		// Bind queue to exchange
		err = ch.QueueBind(
			queueName,                          // queue name
			fmt.Sprintf("reports.%s", country), // routing key
			"warreports",                       // exchange name
			false,                              // no-wait
			nil,                                // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue %s: %w", queueName, err)
		}
	}

	log.Println("[GRPC-SERVER] RabbitMQ initialized successfully")
	return nil
}

// publishToRabbitMQ publica un mensaje a RabbitMQ
func publishToRabbitMQ(country string, message []byte) error {
	mu.Lock()
	defer mu.Unlock()

	routingKey := fmt.Sprintf("reports.%s", country)

	return ch.Publish(
		"warreports", // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message,
		},
	)
}

func main() {
	log.Println("[GRPC-SERVER] Starting gRPC server...")
	log.Printf("[GRPC-SERVER] Listening on %s", grpcPort)

	// Initialize RabbitMQ
	if err := initRabbitMQ(); err != nil {
		log.Fatalf("[GRPC-SERVER] Failed to initialize RabbitMQ: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	// Listen on gRPC port
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("[GRPC-SERVER] Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterWarReportServiceServer(grpcServer, &warReportServer{})

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("[GRPC-SERVER] Shutting down...")
		grpcServer.GracefulStop()
		lis.Close()
		os.Exit(0)
	}()

	log.Printf("[GRPC-SERVER] Server ready on %s", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("[GRPC-SERVER] gRPC serve error: %v", err)
	}
}
