package cron

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

const (
	CRON_SCRIPT   = "/home/pdrpm/SOPES1/LAB_SO1_1S2026/proyecto2/cronjob/crear-containers.sh"
	CRON_INTERVAL = 2 * time.Minute
)

var (
	stopChan chan struct{}
	running  = false
)

// StartCronJob inicia la ejecución manual del cronjob cada 2 minutos en background
func StartCronJob() error {
	if running {
		log.Printf("[cron] Cronjob ya está ejecutándose")
		return nil
	}

	log.Printf("[cron] Iniciando cronjob manual (cada %v)...", CRON_INTERVAL)

	stopChan = make(chan struct{})
	running = true

	// Ejecutar script inmediatamente
	if err := runCronScript(); err != nil {
		log.Printf("[cron] Warning: error en ejecución inicial: %v", err)
	}

	// Iniciar gorrutina que ejecuta el script cada 2 minutos
	go func() {
		ticker := time.NewTicker(CRON_INTERVAL)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := runCronScript(); err != nil {
					log.Printf("[cron] Error ejecutando cronjob: %v", err)
				}
			case <-stopChan:
				log.Printf("[cron] Cronjob detenido")
				return
			}
		}
	}()

	return nil
}

// StopCronJob detiene la ejecución del cronjob
func StopCronJob() error {
	if !running {
		log.Printf("[cron] Cronjob no está ejecutándose")
		return nil
	}

	log.Printf("[cron] Deteniendo cronjob manual...")
	running = false
	close(stopChan)

	return nil
}

// runCronScript ejecuta el script de creación de contenedores
func runCronScript() error {
	log.Printf("[cron] Ejecutando script: %s", CRON_SCRIPT)

	cmd := exec.Command("bash", CRON_SCRIPT)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error al ejecutar cronjob script: %w\noutput: %s", err, string(out))
	}

	log.Printf("[cron] Script ejecutado exitosamente")
	if len(out) > 0 {
		log.Printf("[cron] Output: %s", strings.TrimSpace(string(out)))
	}

	return nil
}

// AddCronJob (deprecated) - mantenido por compatibilidad
func AddCronJob() error {
	return StartCronJob()
}

// RemoveCronJob (deprecated) - mantenido por compatibilidad
func RemoveCronJob() error {
	return StopCronJob()
}
