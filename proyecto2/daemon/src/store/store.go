package store

import (
	"context"
	"fmt"
	"log"

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
