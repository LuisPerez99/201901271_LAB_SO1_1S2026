#!/bin/bash

echo "Iniciando daemon..."

sudo nohup ./build/daemon > daemon.log 2>&1 &
echo $! > daemon.pid

echo "Daemon iniciado con PID $(cat daemon.pid)"
