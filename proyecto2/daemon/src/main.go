package main

import (
	"daemon-proyecto2/src/compose"
	"daemon-proyecto2/src/cron"
	"daemon-proyecto2/src/db"
	"daemon-proyecto2/src/manager"
	"daemon-proyecto2/src/module"
	"daemon-proyecto2/src/parser"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	PROC_FILE     = "/proc/continfo_pr2_so1_201901271"
	LOOP_INTERVAL = 20 * time.Second
)

func main() {
	log.Println("[daemon] Iniciando servicio...")

	// Iniciar docker compose para levantar los servicios de valkey/grafana
	if err := compose.RunCompose(); err != nil {
		log.Fatalf("[daemon] Error al iniciar servicios: %v", err)
	}

	// Instalar cronjob del sistema (cronie)
	if err := cron.StartCronJob(); err != nil {
		log.Fatalf("[daemon] Error al instalar cronjob: %v", err)
	}

	// Cargar modulo del kernel para obtener información de los containers
	if err := module.LoadModule(); err != nil {
		log.Fatalf("[daemon] Error al cargar módulo del kernel: %v", err)
	}

	// Conectar a valkey
	valkeyClient, err := db.NewClient("localhost:6379")
	if err != nil {
		log.Fatalf("[daemon] Error al conectar a valkey: %v", err)
	}
	defer valkeyClient.Close()

	log.Println("[daemon] Servicio iniciado, monitoreando contenedores...")

	// Manejar la señal de interrupción para limpiar recursos
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(LOOP_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := runIteration(valkeyClient); err != nil {
				log.Printf("[daemon] Error en iteración: %v", err)
			}
		case sig := <-sigChan:
			log.Printf("[daemon] Recibida señal %s, terminando servicio...", sig)
			shutdown()
			return
		}
	}
}

func runIteration(valkeyClient *db.Client) error {
	cronLogLines, err := cron.ReadNewExecutionLogs()
	if err != nil {
		log.Printf("[daemon] Advertencia: no se pudieron leer logs del cronjob: %v", err)
	} else {
		for _, line := range cronLogLines {
			log.Printf("[cronjob] %s", line)
		}
	}

	log.Println("[daemon] Leyendo archivo proc...")

	memoryInfo, err := parser.ReadSystemMemoryInfo(PROC_FILE)
	if err != nil {
		log.Printf("[daemon] Advertencia: no se pudo leer memoria del sistema: %v", err)
	}

	// Leer la información de los contenedores desde el archivo proc
	containers, err := parser.ReadContainerInfo(PROC_FILE)
	if err != nil {
		return fmt.Errorf("error al leer el archivo proc: %w", err)
	}

	managedContainers, deletionStats, err := manager.Manage(containers)
	if err != nil {
		return fmt.Errorf("error al manejar contenedores: %w", err)
	}

	log.Printf("[daemon] Guardando estadísticas de %d contenedores del cronjob (eliminados: %d)...",
		len(managedContainers), deletionStats.TotalCount)

	// Guardar logs en Valkey
	if err := valkeyClient.SaveContainerStats(managedContainers, memoryInfo, deletionStats); err != nil {
		return fmt.Errorf("error al guardar estadísticas en Valkey: %w", err)
	}

	return nil
}

func shutdown() {
	log.Println("[daemon] Deteniendo cronjob...")
	if err := cron.StopCronJob(); err != nil {
		log.Printf("[daemon] Error al detener cronjob: %v", err)
	}

	if err := manager.RemoveAllCronJobContainers(); err != nil {
		log.Printf("[daemon] Error al eliminar contenedores del cronjob: %v", err)
	}

	if err := module.UnloadModule(); err != nil {
		log.Printf("[daemon] Error al descargar módulo del kernel: %v", err)
	}

	log.Println("[daemon] Servicio terminado.")
}
