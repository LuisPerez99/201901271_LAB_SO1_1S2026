package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
)

var (
	rabbitmqAddr  = os.Getenv("RABBITMQ_ADDR")
	rabbitmqUser  = os.Getenv("RABBITMQ_USER")
	rabbitmqPass  = os.Getenv("RABBITMQ_PASS")
	rabbitmqVhost = os.Getenv("RABBITMQ_VHOST")

	valkeyAddr = os.Getenv("VALKEY_ADDR")
	valkeyPass = os.Getenv("VALKEY_PASS")

	consumerCountry = os.Getenv("CONSUMER_COUNTRY")

	// Metrics
	mu                sync.Mutex
	messagesProcessed int64
	messagesFailed    int64
)

func init() {
	if rabbitmqAddr == "" {
		rabbitmqAddr = "rabbitmq:5672"
	}
	if rabbitmqUser == "" {
		rabbitmqUser = "guest"
	}
	if rabbitmqPass == "" {
		rabbitmqPass = "guest"
	}
	if rabbitmqVhost == "" {
		rabbitmqVhost = "warreports"
	}
	if valkeyAddr == "" {
		valkeyAddr = "valkey-svc:6379"
	}
	if consumerCountry == "" {
		consumerCountry = "usa"
	}
}

// Report estructura para procesar reportes
type Report struct {
	Country         string `json:"country"`
	WarplanesInAir  int32  `json:"warplanes_in_air"`
	WarshipsInWater int32  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
	ReceivedAt      string `json:"received_at,omitempty"`
}

func main() {
	log.Println("[CONSUMER] Starting RabbitMQ consumer...")
	log.Printf("[CONSUMER] Country: %s", consumerCountry)
	log.Printf("[CONSUMER] RabbitMQ: %s", rabbitmqAddr)
	log.Printf("[CONSUMER] Valkey: %s", valkeyAddr)

	// Connect to RabbitMQ
	vhost := strings.TrimPrefix(rabbitmqVhost, "/")
	dsn := fmt.Sprintf("amqp://%s:%s@%s/%s", rabbitmqUser, rabbitmqPass, rabbitmqAddr, vhost)
	conn, err := amqp.Dial(dsn)
	if err != nil {
		log.Fatalf("[CONSUMER] Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("[CONSUMER] Failed to open channel: %v", err)
	}
	defer ch.Close()

	// Connect to Valkey
	rdb := redis.NewClient(&redis.Options{
		Addr:     valkeyAddr,
		Password: valkeyPass,
	})
	defer rdb.Close()

	// Test Valkey connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = rdb.Ping(ctx).Err()
	cancel()
	if err != nil {
		log.Fatalf("[CONSUMER] Failed to connect to Valkey: %v", err)
	}
	log.Println("[CONSUMER] Connected to Valkey")

	// Declare queue
	queueName := fmt.Sprintf("reports.%s", consumerCountry)
	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Fatalf("[CONSUMER] Failed to declare queue: %v", err)
	}

	// Setup consumer
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("[CONSUMER] Failed to register consumer: %v", err)
	}

	log.Printf("[CONSUMER] Started consuming from %s", queueName)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("[CONSUMER] Shutting down...")
		log.Printf("[CONSUMER] Processed: %d, Failed: %d", messagesProcessed, messagesFailed)
		os.Exit(0)
	}()

	// Process messages
	for d := range msgs {
		var report Report
		if err := json.Unmarshal(d.Body, &report); err != nil {
			log.Printf("[CONSUMER] Failed to unmarshal message: %v", err)
			mu.Lock()
			messagesFailed++
			mu.Unlock()
			d.Nack(false, true) // negative ack, requeue
			continue
		}

		// Store in Valkey
		key := fmt.Sprintf("%s:report:%s", consumerCountry, time.Now().Format("20060102150405"))
		report.ReceivedAt = time.Now().UTC().Format(time.RFC3339)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		reportJSON, _ := json.Marshal(report)

		err = storeDashboardData(ctx, rdb, key, report, reportJSON)
		cancel()

		if err != nil {
			log.Printf("[CONSUMER] Failed to store in Valkey: %v", err)
			mu.Lock()
			messagesFailed++
			mu.Unlock()
			d.Nack(false, true)
			continue
		}

		mu.Lock()
		messagesProcessed++
		mu.Unlock()

		log.Printf("[CONSUMER] Stored report: key=%s", key)
		d.Ack(false) // acknowledge
	}
}

func storeDashboardData(ctx context.Context, rdb *redis.Client, key string, report Report, reportJSON []byte) error {
	parsedTimestamp, err := time.Parse(time.RFC3339Nano, report.Timestamp)
	if err != nil {
		parsedTimestamp = time.Now().UTC()
	}

	countryKey := fmt.Sprintf("dashboard:country:%s", report.Country)
	globalPlanesValuesKey := "dashboard:global:planes:values"
	globalShipsValuesKey := "dashboard:global:ships:values"
	globalPlanesFreqKey := "dashboard:global:planes:freq"
	globalShipsFreqKey := "dashboard:global:ships:freq"
	timelineKey := fmt.Sprintf("%s:timeline", countryKey)

	pipe := rdb.TxPipeline()
	pipe.Set(ctx, key, string(reportJSON), 7*24*time.Hour)
	pipe.Set(ctx, fmt.Sprintf("%s:raw:%s", countryKey, parsedTimestamp.Format(time.RFC3339Nano)), string(reportJSON), 7*24*time.Hour)
	pipe.Incr(ctx, fmt.Sprintf("%s:reports_total", countryKey))
	pipe.Incr(ctx, "dashboard:global:reports_total")
	pipe.HIncrBy(ctx, fmt.Sprintf("%s:planes:sum", countryKey), "total", int64(report.WarplanesInAir))
	pipe.HIncrBy(ctx, fmt.Sprintf("%s:ships:sum", countryKey), "total", int64(report.WarshipsInWater))
	pipe.HIncrBy(ctx, globalPlanesFreqKey, strconv.FormatInt(int64(report.WarplanesInAir), 10), 1)
	pipe.HIncrBy(ctx, globalShipsFreqKey, strconv.FormatInt(int64(report.WarshipsInWater), 10), 1)
	pipe.HIncrBy(ctx, fmt.Sprintf("%s:planes:freq", countryKey), strconv.FormatInt(int64(report.WarplanesInAir), 10), 1)
	pipe.HIncrBy(ctx, fmt.Sprintf("%s:ships:freq", countryKey), strconv.FormatInt(int64(report.WarshipsInWater), 10), 1)
	pipe.ZAdd(ctx, globalPlanesValuesKey, redis.Z{
		Score:  float64(report.WarplanesInAir),
		Member: fmt.Sprintf("%s:%s:planes:%d", parsedTimestamp.Format(time.RFC3339Nano), report.Country, report.WarplanesInAir),
	})
	pipe.ZAdd(ctx, globalShipsValuesKey, redis.Z{
		Score:  float64(report.WarshipsInWater),
		Member: fmt.Sprintf("%s:%s:ships:%d", parsedTimestamp.Format(time.RFC3339Nano), report.Country, report.WarshipsInWater),
	})
	pipe.ZAdd(ctx, timelineKey, redis.Z{
		Score: float64(parsedTimestamp.UnixNano()),
		Member: fmt.Sprintf(
			"country=%s|planes=%d|ships=%d|ts=%s",
			report.Country,
			report.WarplanesInAir,
			report.WarshipsInWater,
			parsedTimestamp.Format(time.RFC3339Nano),
		),
	})

	_, err = pipe.Exec(ctx)
	return err
}
