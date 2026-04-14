package cron

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	CRON_SCRIPT   = "/home/pdrpm/SOPES1/LAB_SO1_1S2026/proyecto2/cronjob/crear-containers.sh"
	CRON_LOG_FILE = "/var/log/cronjob.log"
	CRON_D_DIR    = "/etc/cron.d"
	CRON_JOB_NAME = "pr2-proyecto2-cronjob"
	CRON_SCHEDULE = "*/2 * * * *" // Cada 2 minutos
)

var (
	logOffsetMu sync.Mutex
	logOffset   int64 = -1
)

// StartCronJob instala el cronjob en el demonio cron del sistema (cronie)
func StartCronJob() error {
	log.Printf("[cron] Instalando cronjob del sistema cada 2 minutos...")

	// Crear contenido del archivo cron
	cronJobPath := filepath.Join(CRON_D_DIR, CRON_JOB_NAME)
	cronContent := fmt.Sprintf("# Proyecto 2 - Cronjob para crear contenedores\n# DO NOT EDIT - Generado por daemon\n%s root bash %s\n",
		CRON_SCHEDULE, CRON_SCRIPT)

	// Verificar si el archivo ya existe
	if _, err := os.Stat(cronJobPath); err == nil {
		log.Printf("[cron] Cronjob ya está instalado en %s", cronJobPath)
		return nil
	}

	// Crear el archivo de cronjob (requiere sudo)
	cmd := exec.Command("sudo", "-n", "tee", cronJobPath)
	cmd.Stdin = strings.NewReader(cronContent)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al instalar cronjob (requiere sudo NOPASSWD): %v, salida: %s", err, string(output))
	}

	// Establecer permisos del archivo
	cmd = exec.Command("sudo", "-n", "chmod", "644", cronJobPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al establecer permisos del cronjob: %v, salida: %s", err, string(output))
	}

	log.Printf("[cron] Cronjob instalado exitosamente en %s", cronJobPath)
	return nil
}

// StopCronJob desinstala el cronjob del demonio cron del sistema
func StopCronJob() error {
	log.Printf("[cron] Desinstalando cronjob del sistema...")

	cronJobPath := filepath.Join(CRON_D_DIR, CRON_JOB_NAME)

	// Verificar si el archivo existe
	if _, err := os.Stat(cronJobPath); err != nil {
		log.Printf("[cron] Cronjob no encontrado en %s", cronJobPath)
		return nil
	}

	// Eliminar el archivo de cronjob (requiere sudo)
	cmd := exec.Command("sudo", "-n", "rm", "-f", cronJobPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error al desinstalar cronjob: %v, salida: %s", err, string(output))
	}

	log.Printf("[cron] Cronjob desinstalado exitosamente")
	return nil
}

// ReadNewExecutionLogs devuelve las nuevas lineas del log del cronjob desde la
// ultima lectura. Si el archivo se trunca, reinicia automaticamente el offset.
func ReadNewExecutionLogs() ([]string, error) {
	logOffsetMu.Lock()
	defer logOffsetMu.Unlock()

	file, err := os.Open(CRON_LOG_FILE)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error al abrir log del cronjob: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error al obtener metadata del log del cronjob: %w", err)
	}

	if logOffset < 0 || logOffset > info.Size() {
		logOffset = info.Size()
		return nil, nil
	}

	if _, err := file.Seek(logOffset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("error al posicionar lectura del log del cronjob: %w", err)
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error al leer log del cronjob: %w", err)
	}

	newOffset, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf("error al actualizar offset del log del cronjob: %w", err)
	}
	logOffset = newOffset

	return lines, nil
}
