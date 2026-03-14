package compose

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Función para retornar la ruta del directorio del compose de grafana y valkey
func composeDir() string {
	_, file, _, _ := runtime.Caller(0)

	base := filepath.Dir(file)
	return filepath.Join(base, "..", "..", "..", "grafana")
}

// Correr el docker-compose para levantar los servicios de grafana y valkey
func RunCompose() error {
	path := composeDir()
	log.Printf("[grafana-valkey] Iniciando servicios en %s", path)

	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = path

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error al iniciar los servicios: %v, salida: %s", err, string(out))
	}

	log.Printf("[grafana-valkey] Servicios iniciados:\n%s", string(out))
	return nil
}

// Detener los servicios de grafana y valkey usando docker-compose
func StopCompose() error {
	path := composeDir()
	log.Printf("[grafana-valkey] Deteniendo servicios en %s", path)

	cmd := exec.Command("docker", "compose", "down")
	cmd.Dir = path

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error al detener los servicios: %v, salida: %s", err, string(out))
	}

	log.Printf("[grafana-valkey] Servicios detenidos:\n%s", string(out))
	return nil
}
