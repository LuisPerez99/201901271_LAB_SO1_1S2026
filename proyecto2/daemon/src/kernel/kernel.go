package kernel

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"
)

const MODULE_NAME = "continfo_pr2_so1_201901271"

// Directorio del módulo del kernel
func modulePath() string {
	_, file, _, _ := runtime.Caller(0)
	base := filepath.Dir(file)
	return filepath.Join(base, "..", "..", "..", "modulo-kernel", "build", MODULE_NAME+".ko")
}

// Cargar el módulo del kernel
func LoadModule() error {
	path := modulePath()
	log.Printf("[kernel] Cargando el módulo desde %s", path)

	// Verificar si el módulo ya está cargado
	out, _ := exec.Command("lsmod").Output()
	if contains(string(out), MODULE_NAME) {
		log.Printf("[kernel] El módulo ya está cargado")
		return nil
	}

	cmd := exec.Command("sudo", "insmod", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error al cargar el módulo: %v, salida: %s", err, string(output))
	}

	log.Printf("[kernel] Módulo cargado exitosamente")
	return nil
}

// Descargar el módulo del kernel
func UnloadModule() error {
	log.Printf("[kernel] Descargando el módulo %s", MODULE_NAME)

	cmd := exec.Command("sudo", "rmmod", MODULE_NAME)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error al descargar el módulo: %v, salida: %s", err, string(output))
	}

	log.Printf("[kernel] Módulo descargado exitosamente")
	return nil
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || (len(s) > len(substr) &&
			(s[:len(substr)] == substr || contains(s[1:], substr))))
}
