package db

import (
	"context"
	"daemon-proyecto2/src/parser"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	STATS_KEY   = "container:stats"
	HISTORY_KEY = "container:history"
	MAX_HISTORY = 100
)

type Client struct {
	rdb *redis.Client
	ctx context.Context
}

// Crear un nuevo cliente de valkey con la dirección del servidor Redis y probar la conexión
func NewClient(addr string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	ctx := context.Background()

	// Probar la conexión a Valkey
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("error al conectar a Valkey en %s: %w", addr, err)
	}

	log.Printf("[store] Conectado a valkey en %s", addr)
	return &Client{rdb: rdb, ctx: ctx}, nil
}

// Guardar las estadísticas actuales de los contenedores en Valkey
func (c *Client) SaveContainerStats(containers []parser.ContainerInfo) error {
	timestamp := time.Now().Unix()

	// Guardar las estadísticas actuales en Valkey
	snapshot := map[string]interface{}{
		"timestamp":  timestamp,
		"containers": containers,
		"total":      len(containers),
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("error al serializar las estadísticas: %w", err)
	}

	if err := c.rdb.Set(c.ctx, STATS_KEY, data, 0).Err(); err != nil {
		return fmt.Errorf("error al guardar las estadísticas en Valkey: %w", err)
	}

	// Guardar el historial de estadísticas
	if err := c.rdb.LPush(c.ctx, HISTORY_KEY, data).Err(); err != nil {
		return fmt.Errorf("error al guardar el historial en Valkey: %w", err)
	}

	// Limitar el historial a MAX_HISTORY entradas
	if err := c.rdb.LTrim(c.ctx, HISTORY_KEY, 0, MAX_HISTORY-1).Err(); err != nil {
		return fmt.Errorf("error al limitar el historial en Valkey: %w", err)
	}

	for _, cont := range containers {
		key := fmt.Sprintf("container:%s:stats", cont.Name)
		stats := map[string]interface{}{
			"timestamp": timestamp,
			"ram":       cont.RAM,
			"vsz":       cont.VSZ,
			"rss":       cont.RSS,
			"cpu":       cont.CPU,
			"is_high":   cont.IsHigh,
		}
		statsData, _ := json.Marshal(stats)
		if err := c.rdb.Set(c.ctx, key, statsData, 24*time.Hour).Err(); err != nil {
			return fmt.Errorf("error al guardar las estadísticas del contenedor %s en Valkey: %w", cont.Name, err)
		}
	}

	log.Printf("[guardar] Estadísticas guardadas en Valkey con timestamp %d", timestamp)
	return nil
}

// Cerrar la conexión con Valkey
func (c *Client) Close() {
	c.rdb.Close()
}
