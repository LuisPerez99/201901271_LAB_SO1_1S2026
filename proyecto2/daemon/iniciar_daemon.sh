#!/bin/bash

# Verificar si existe el binario del daemon y compilar si no existe
if [ ! -f "./build/daemon" ]; then
    echo "Compilando el daemon..."
    go build -o build/daemon ./src
    if [ $? -ne 0 ]; then
        echo "Error al compilar el daemon. Abortando."
        exit 1
    fi
fi

echo "Iniciando daemon..."

sudo nohup ./build/daemon > daemon.log 2>&1 &
echo $! > daemon.pid

echo "Daemon iniciado con PID $(cat daemon.pid)"
