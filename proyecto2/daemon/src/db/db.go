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
	STATS_KEY                 = "container:stats"
	HISTORY_KEY               = "container:history"
	FLAT_HISTORY_KEY          = "container:flat:history"
	SYSTEM_MEMORY_KEY         = "system:memory"
	SYSTEM_MEMORY_HISTORY_KEY = "system:memory:history"
	DELETIONS_HISTORY_KEY     = "container:deletions:history"
	MAX_HISTORY               = 100
	MAX_FLAT_HISTORY          = 1000
)

type DeletionStats struct {
	Timestamp   int64                    `json:"timestamp"`
	TotalCount  int                      `json:"total_count"`
	HighRemoved int                      `json:"high_removed"`
	LowRemoved  int                      `json:"low_removed"`
	Deleted     []map[string]interface{} `json:"deleted"`
}

type FlatContainerMetric struct {
	Timestamp int64   `json:"timestamp"`
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	PID       int     `json:"pid"`
	RAM       uint64  `json:"ram"`
	VSZ       uint64  `json:"vsz"`
	RSS       uint64  `json:"rss"`
	CPU       float64 `json:"cpu"`
	IsHigh    bool    `json:"is_high"`
}

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

	log.Printf("[db] Conectado a valkey en %s", addr)
	return &Client{rdb: rdb, ctx: ctx}, nil
}

// Guardar las estadísticas actuales de los contenedores en Valkey
func (c *Client) SaveContainerStats(containers []parser.ContainerInfo, memory parser.SystemMemoryInfo, deletionStats *DeletionStats) error {
	timestamp := time.Now().Unix()

	// Guardar las estadísticas actuales en Valkey
	snapshot := map[string]interface{}{
		"timestamp":     timestamp,
		"containers":    containers,
		"total":         len(containers),
		"system_memory": memory,
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("error al serializar las estadísticas: %w", err)
	}

	if err := c.rdb.Set(c.ctx, STATS_KEY, data, 0).Err(); err != nil {
		return fmt.Errorf("error al guardar las estadísticas en Valkey: %w", err)
	}

	memoryData, err := json.Marshal(map[string]interface{}{
		"timestamp": timestamp,
		"total_mb":  memory.TotalMB,
		"free_mb":   memory.FreeMB,
		"used_mb":   memory.UsedMB,
	})
	if err != nil {
		return fmt.Errorf("error al serializar la memoria del sistema: %w", err)
	}

	if err := c.rdb.Set(c.ctx, SYSTEM_MEMORY_KEY, memoryData, 0).Err(); err != nil {
		return fmt.Errorf("error al guardar la memoria del sistema en Valkey: %w", err)
	}

	if err := c.SaveSystemMemoryHistory(memoryData); err != nil {
		return err
	}

	// Guardar el historial de estadísticas
	if err := c.rdb.LPush(c.ctx, HISTORY_KEY, data).Err(); err != nil {
		return fmt.Errorf("error al guardar el historial en Valkey: %w", err)
	}

	// Limitar el historial a MAX_HISTORY entradas
	if err := c.rdb.LTrim(c.ctx, HISTORY_KEY, 0, MAX_HISTORY-1).Err(); err != nil {
		return fmt.Errorf("error al limitar el historial en Valkey: %w", err)
	}

	if err := c.SaveFlatContainerHistory(timestamp, containers); err != nil {
		return err
	}

	// Guardar estadísticas de eliminaciones
	if deletionStats != nil && deletionStats.TotalCount > 0 {
		deletionData, err := json.Marshal(deletionStats)
		if err != nil {
			log.Printf("[db] Advertencia: error al serializar estadísticas de eliminaciones: %v", err)
		} else {
			if err := c.rdb.LPush(c.ctx, DELETIONS_HISTORY_KEY, deletionData).Err(); err != nil {
				log.Printf("[db] Advertencia: error al guardar historial de eliminaciones: %v", err)
			}

			if err := c.rdb.LTrim(c.ctx, DELETIONS_HISTORY_KEY, 0, MAX_HISTORY-1).Err(); err != nil {
				log.Printf("[db] Advertencia: error al limitar historial de eliminaciones: %v", err)
			}
		}
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

	deletedCount := 0
	if deletionStats != nil {
		deletedCount = deletionStats.TotalCount
	}

	log.Printf("[db] Estadísticas guardadas en Valkey (eliminados: %d)", deletedCount)
	return nil
}

// SaveFlatContainerHistory guarda una fila por contenedor para facilitar agregaciones históricas en Grafana.
func (c *Client) SaveFlatContainerHistory(timestamp int64, containers []parser.ContainerInfo) error {
	for _, cont := range containers {
		row := FlatContainerMetric{
			Timestamp: timestamp,
			ID:        cont.ID,
			Name:      cont.Name,
			PID:       cont.PID,
			RAM:       cont.RAM,
			VSZ:       cont.VSZ,
			RSS:       cont.RSS,
			CPU:       cont.CPU,
			IsHigh:    cont.IsHigh,
		}

		rowData, err := json.Marshal(row)
		if err != nil {
			return fmt.Errorf("error al serializar fila flat del contenedor %s: %w", cont.Name, err)
		}

		if err := c.rdb.LPush(c.ctx, FLAT_HISTORY_KEY, rowData).Err(); err != nil {
			return fmt.Errorf("error al guardar fila flat del contenedor %s en Valkey: %w", cont.Name, err)
		}
	}

	if err := c.rdb.LTrim(c.ctx, FLAT_HISTORY_KEY, 0, MAX_FLAT_HISTORY-1).Err(); err != nil {
		return fmt.Errorf("error al limitar el historial flat en Valkey: %w", err)
	}

	return nil
}

// SaveSystemMemoryHistory guarda un punto histórico de memoria del sistema en Valkey.
func (c *Client) SaveSystemMemoryHistory(memoryData []byte) error {
	if err := c.rdb.LPush(c.ctx, SYSTEM_MEMORY_HISTORY_KEY, memoryData).Err(); err != nil {
		return fmt.Errorf("error al guardar el historial de memoria del sistema en Valkey: %w", err)
	}

	if err := c.rdb.LTrim(c.ctx, SYSTEM_MEMORY_HISTORY_KEY, 0, MAX_HISTORY-1).Err(); err != nil {
		return fmt.Errorf("error al limitar el historial de memoria del sistema en Valkey: %w", err)
	}

	return nil
}

// Cerrar la conexión con Valkey
func (c *Client) Close() {
	c.rdb.Close()
}
