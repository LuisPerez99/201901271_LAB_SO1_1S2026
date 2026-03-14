package cron

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

const (
	CRON_SCRIPT = "/home/pdrpm/SOPES1/LAB_SO1_1S2026/proyecto2/cronjob/crear-containers.sh"
	CRON_ENTRY  = "*/2 * * * * " + CRON_SCRIPT + " >> /tmp/cronjob.log 2>&1"
)

// Agregar la tarea al crontab
func AddCronJob() error {
	log.Printf("[cron] Agregando tarea al crontab: %s", CRON_ENTRY)

	// Obtener el crontab actual
	out, err := exec.Command("crontab", "-l").Output()

	// Verificar si la tarea ya existe
	if strings.Contains(string(out), CRON_SCRIPT) {
		log.Printf("[cron] La tarea ya existe en el crontab")
		return nil
	}

	// Agregar la nueva tarea al crontab
	newCrontab := string(out) + CRON_ENTRY + "\n"

	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)

	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error al agregar la tarea al crontab: %v, salida: %s", err, string(out))
	}

	log.Printf("[cron] Tarea agregada al crontab exitosamente")
	return nil
}

// Eliminar la tarea del crontab
func RemoveCronJob() error {
	log.Printf("[cron] Eliminando tarea del crontab: %s", CRON_ENTRY)

	// Obtener el crontab actual
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return fmt.Errorf("error al obtener el crontab: %v, salida: %s", err, string(out))
	}

	// Eliminar la tarea del crontab
	lines := strings.Split(string(out), "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.Contains(line, CRON_SCRIPT) {
			filtered = append(filtered, line)
		}
	}

	newCrontab := strings.Join(filtered, "\n")
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)

	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error al eliminar la tarea del crontab: %v, salida: %s", err, string(out))
	}

	log.Printf("[cron] Tarea eliminada del crontab exitosamente")
	return nil
}
