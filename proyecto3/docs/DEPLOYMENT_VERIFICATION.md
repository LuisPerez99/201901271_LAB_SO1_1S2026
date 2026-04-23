# Verificación Operativa - Proyecto3 M.U.M.N.K8s

Este documento proporciona una guía paso a paso para verificar que todos los componentes del proyecto están funcionando correctamente después del deployment.

**Carnet**: 201901271  
**País**: USA  
**Namespace**: proyecto3

---

## Fase 0: Pre-Verificación

### 0.1 Verificar Cluster GKE

```bash
# Verificar conexión a cluster
kubectl cluster-info

# Verificar nodos
kubectl get nodes
# Esperado: 1+ nodos en estado Ready

# Verificar versión de Kubernetes
kubectl version --short

# Verificar que el namespace existe
kubectl get namespace proyecto3
# Esperado: STATUS = Active
```

### 0.2 Verificar Zot Registry

```bash
export ZOT_HOST="<IP_PUBLICA_VM>"
export ZOT_USER="admin"
export ZOT_PASS="admin123"

# Verificar conectividad a Zot HTTPS
curl -k -u ${ZOT_USER}:${ZOT_PASS} https://${ZOT_HOST}:5000/v2/
# Esperado: 200 OK

# Verificar catálogo de imágenes
curl -k -u ${ZOT_USER}:${ZOT_PASS} https://${ZOT_HOST}:5000/v2/_catalog
# Esperado: 
# {
#   "repositories": [
#     "rust-api",
#     "api-client", 
#     "grpc-server",
#     "go-consumer"
#   ]
# }

# Verificar tags de cada imagen
for img in rust-api api-client grpc-server go-consumer; do
    echo "=== $img ==="
    curl -k -u ${ZOT_USER}:${ZOT_PASS} https://${ZOT_HOST}:5000/v2/${img}/tags/list
done
# Esperado: Todos deben tener tag "201901271"
```

---

## Fase 1: Verificar Deployments Básicos

### 1.1 Verificar Namespace

```bash
kubectl describe ns proyecto3
```

### 1.2 Verificar RabbitMQ

```bash
# Ver pod de RabbitMQ
kubectl get pods -n proyecto3 -l app=rabbitmq

# Ver descripción detallada
kubectl describe pod -n proyecto3 -l app=rabbitmq

# Verificar logs
kubectl logs -n proyecto3 -l app=rabbitmq --tail=50

# Verificar que el PVC está montado
kubectl get pvc -n proyecto3
# Esperado: rabbitmq-pvc en status Bound

# Verificar servicio
kubectl get svc -n proyecto3 -l app=rabbitmq
# Esperado: rabbitmq service con puerto 5672 (amqp) y 15672 (management)

# Ejecutar comando de diagnostics dentro del pod
kubectl exec -it -n proyecto3 -l app=rabbitmq -- rabbitmq-diagnostics status
# Esperado: Información de status del broker
```

### 1.3 Verificar Rust API

```bash
# Ver pod
kubectl get pods -n proyecto3 -l app=rust-api

# Ver descripción
kubectl describe pod -n proyecto3 -l app=rust-api

# Ver logs
kubectl logs -n proyecto3 -l app=rust-api --tail=50
# Esperado: "[INFO] Listening on 0.0.0.0:3000"

# Verificar servicio
kubectl get svc -n proyecto3 -l app=rust-api
# Esperado: rust-api service ClusterIP :3000

# Health check
kubectl exec -it -n proyecto3 -l app=rust-api -- curl -s localhost:3000/health
# Esperado: {"status": "ok"}
```

### 1.4 Verificar Go API Client (2-container)

```bash
# Ver pod con ambos contenedores
kubectl get pods -n proyecto3 -l app=api-client -o wide

# Verificar que tiene 2 contenedores
kubectl describe pod -n proyecto3 -l app=api-client | grep "Containers:"
# Esperado: 2/2 Ready

# Ver logs del contenedor principal
kubectl logs -n proyecto3 -l app=api-client -c api-client

# Ver logs del sidecar
kubectl logs -n proyecto3 -l app=api-client -c api-client-sidecar

# Verificar servicio
kubectl get svc -n proyecto3 -l app=api-client
# Esperado: api-client service ClusterIP :8080
```

### 1.5 Verificar Go gRPC Server (2-container)

```bash
# Ver pod con ambos contenedores
kubectl get pods -n proyecto3 -l app=grpc-server

# Verificar que tiene 2 contenedores
kubectl describe pod -n proyecto3 -l app=grpc-server | grep -A5 "Containers:"

# Ver logs del contenedor principal
kubectl logs -n proyecto3 -l app=grpc-server -c grpc-server
# Esperado: "[GRPC-SERVER] RabbitMQ initialized successfully"

# Ver logs del sidecar
kubectl logs -n proyecto3 -l app=grpc-server -c grpc-server-sidecar

# Verificar servicio
kubectl get svc -n proyecto3 -l app=grpc-server
# Esperado: grpc-server service ClusterIP :50051
```

---

## Fase 2: Verificar KubeVirt Components

### 2.1 Verificar Valkey VM

```bash
# Ver VM
kubectl get vms -n proyecto3
# Esperado: valkey-vm en estado Running

# Ver VirtualMachine details
kubectl describe vm valkey-vm -n proyecto3

# Ver pod del VMI
kubectl get vmis -n proyecto3
# Esperado: valkey-vm en estado Running

# Ver servicio
kubectl get svc -n proyecto3 valkey-svc
# Esperado: ClusterIP :6379

# Port-forward para acceso local
kubectl port-forward -n proyecto3 svc/valkey-svc 6379:6379 &
PF_PID=$!

# Verificar conectividad con redis-cli
(echo "ping"; echo "keys *"; echo "quit") | nc localhost 6379
# O si tienes redis-cli:
# redis-cli -h localhost -p 6379 ping
# Esperado: PONG

# Matar port-forward
kill $PF_PID
```

### 2.2 Verificar Grafana VM

```bash
# Ver VM
kubectl get vms -n proyecto3 -l app=grafana

# Ver VirtualMachine details
kubectl describe vm grafana-vm -n proyecto3

# Ver VMI running
kubectl get vmis -n proyecto3 -l app=grafana

# Ver servicio
kubectl get svc -n proyecto3 grafana-svc
# Esperado: ClusterIP :3000

# Port-forward para acceso web
kubectl port-forward -n proyecto3 svc/grafana-svc 3000:3000 &
PF_PID=$!

# Verificar acceso HTTP
curl -s http://localhost:3000/api/health
# Esperado: {"database":"ok","status":"ok"}

# Verificar datasource provisionado
curl -s http://localhost:3000/api/datasources | jq .
# Esperado: Datasource "valkey-svc" de tipo redis

# Obtener API token si es necesario
ADMIN_PASSWORD="admin123"
curl -s -X POST http://localhost:3000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"user":"admin","password":"'${ADMIN_PASSWORD}'"}'
# Guardará token para requests posteriores

# Matar port-forward
kill $PF_PID
```

---

## Fase 3: Verificar Consumer

### 3.1 Verificar Go Consumer Pod

```bash
# Ver pod
kubectl get pods -n proyecto3 -l app=consumer

# Ver descripción
kubectl describe pod -n proyecto3 -l app=consumer

# Ver logs (consume de reports.usa)
kubectl logs -n proyecto3 -l app=consumer --tail=100
# Esperado:
# [CONSUMER] Connecting to RabbitMQ...
# [CONSUMER] Connected to RabbitMQ
# [CONSUMER] Subscribing to reports.usa
# [CONSUMER] Message received: {...}
# [CONSUMER] Stored in Valkey
```

---

## Fase 4: Verificar Gateway API

### 4.1 Verificar Gateway y HTTPRoute

```bash
# Ver Gateway
kubectl get gateway -n proyecto3
# Esperado: gateway-proyecto3 en estado Ready

# Ver HTTPRoute
kubectl get httproute -n proyecto3
# Esperado: api-route con 1 parent: gateway-proyecto3

# Ver detalles del HTTPRoute
kubectl describe httproute api-route -n proyecto3
# Esperado:
# Rules:
#   Path Type: PathPrefix
#   Matches:
#     Path: /grpc-201901271
#     Path: /health
# Backends:
#   Service: rust-api

# Ver GatewayClass
kubectl get gatewayclass
# Esperado: gke (por defecto en GKE)

# Obtener IP pública del Gateway
kubectl get gateway -n proyecto3 -o jsonpath='{.items[0].status.addresses[0].value}'
# O más detallado:
kubectl describe gateway -n proyecto3
# Esperado: addresses section con IP pública
```

### 4.2 Verificar Conectividad a través del Gateway

```bash
export GATEWAY_IP=$(kubectl get gateway -n proyecto3 -o jsonpath='{.items[0].status.addresses[0].value}')

# Health check
curl http://${GATEWAY_IP}/health
# Esperado: {"status": "ok"}

# Test de reporte (requiere esperar a que esté listo)
curl -X POST http://${GATEWAY_IP}/grpc-201901271 \
  -H "Content-Type: application/json" \
  -d '{
    "country": "USA",
    "warplanes_in_air": 15,
    "warships_in_water": 8,
    "timestamp": "2026-04-22T12:00:00Z"
  }'
# Esperado: 200 OK, el reporte se propaga por toda la cadena
```

---

## Fase 5: Verificar HPA (Horizontal Pod Autoscaler)

### 5.1 Verificar configuración de HPA

```bash
# Ver HPA
kubectl get hpa -n proyecto3
# Esperado: rust-api-hpa

# Ver detalles
kubectl describe hpa rust-api-hpa -n proyecto3
# Esperado:
# Reference: Deployment/rust-api
# Targets: 30% CPU, 70% Memory
# Min replicas: 1, Max replicas: 3
# Current replicas: 1

# Monitorear en tiempo real (en otra terminal)
kubectl get hpa -n proyecto3 -w
```

### 5.2 Generar carga con Locust y monitorear escalado

```bash
# Terminal 1: Ejecutar Locust
cd locust
python -m locust -f locustfile.py --host=http://${GATEWAY_IP} --users 100 --spawn-rate 10

# Terminal 2: Monitorear HPA y pods
watch 'kubectl get hpa,pods -n proyecto3'

# Esperado:
# 1. Inicialmente CURRENT = 1 replica
# 2. CPU % sube por encima de 30%
# 3. Después de ~1 min, DESIRED sube a 2-3
# 4. CURRENT llega a 3 replicas
```

---

## Fase 6: Verificar Métricas en Grafana

### 6.1 Conectar a Grafana

```bash
# Port-forward
kubectl port-forward -n proyecto3 svc/grafana-svc 3000:3000

# Abrir navegador: http://localhost:3000
# Login: admin / admin123
# Expected: Dashboard vacío inicialmente
```

### 6.2 Crear Dashboard de Métricas

En Grafana, crear nuevo Dashboard con panels:

```
Panel 1: Total Reports Globales
Query: GET dashboard:global:reports_total

Panel 2: Reports por País
Query: MGET dashboard:country:USA:reports_total dashboard:country:RUS:reports_total ...

Panel 3: Distribución de Aviones
Query: HGETALL dashboard:global:planes:freq

Panel 4: Distribución de Naves
Query: HGETALL dashboard:global:ships:freq

Panel 5: Timeline por País
Query: ZRANGE USA:timeline 0 -1 WITHSCORES
```

### 6.3 Verificar Data Flow

```bash
# Port-forward a Valkey
kubectl port-forward -n proyecto3 svc/valkey-svc 6379:6379

# Conectar con redis-cli
redis-cli -h localhost -p 6379

# Verificar claves de dashboard
KEYS dashboard:*
# Esperado:
# dashboard:global:reports_total
# dashboard:global:planes:freq
# dashboard:global:ships:freq
# dashboard:country:usa:reports_total
# ...

# Ver valor de total reports
GET dashboard:global:reports_total
# Esperado: Número > 0 si el consumer procesó reportes

# Ver timeline de USA
ZCOUNT USA:timeline -inf +inf
# Esperado: Número de eventos en timeline
```

---

## Fase 7: Pruebas de Carga Completas

### 7.1 Ejecutar Locust

```bash
cd locust

# Verificar dependencias
pip install -r requirements.txt

# Ejecutar load test
python -m locust -f locustfile.py \
  --host=http://${GATEWAY_IP} \
  --users 50 \
  --spawn-rate 5 \
  --run-time 5m \
  --headless

# Esperado:
# - Requests exitosos al endpoint /grpc-201901271
# - Response times < 500ms en promedio
# - < 1% de error rate
# - RPS (requests per second) escalando con número de users
```

### 7.2 Monitorear durante prueba

```bash
# Terminal 1: Locust (arriba)

# Terminal 2: Monitorear Pods
kubectl get pods -n proyecto3 -w

# Terminal 3: Monitorear HPA
kubectl get hpa -n proyecto3 -w

# Terminal 4: Ver logs de Consumer
kubectl logs -n proyecto3 -f -l app=consumer

# Terminal 5: Monitorear RabbitMQ
kubectl exec -it -n proyecto3 -l app=rabbitmq -- \
  rabbitmq-diagnostics queue_memory_breakdown

# Terminal 6: Monitorear Valkey
kubectl port-forward -n proyecto3 svc/valkey-svc 6379:6379
# En otra sub-terminal:
redis-cli -h localhost -p 6379 --stat
```

---

## Fase 8: Verificación de Durabilidad

### 8.1 Verificar Persistencia en RabbitMQ

```bash
# Ver estado de persistencia
kubectl exec -it -n proyecto3 -l app=rabbitmq -- \
  rabbitmq-diagnostics environment

# Ver memoria usada por mensajes
kubectl exec -it -n proyecto3 -l app=rabbitmq -- \
  rabbitmq-diagnostics queues_memory

# Simular fallo y recuperación
# 1. Matar pod de RabbitMQ
kubectl delete pod -n proyecto3 -l app=rabbitmq

# 2. Esperar a que K8s lo recree (15-30 segundos)
kubectl get pods -n proyecto3 -l app=rabbitmq -w

# 3. Verificar que mensajes persisten
kubectl exec -it -n proyecto3 -l app=rabbitmq -- \
  rabbitmq-diagnostics queues
# Esperado: reports.usa y otras queues con mensajes sin procesar
```

### 8.2 Verificar Persistencia en Valkey

```bash
# Conectar a Valkey
kubectl port-forward -n proyecto3 svc/valkey-svc 6379:6379

# Ver tamaño de DB
redis-cli -h localhost -p 6379 DBSIZE

# Ver keys
redis-cli -h localhost -p 6379 KEYS "*"

# Simular fallo de VM:
# 1. Deletear VMI (no VM, para preservar disk)
kubectl delete vmi valkey-vm -n proyecto3

# 2. Esperar a que se recree desde VM
kubectl get vmis -n proyecto3 -w

# 3. Verificar que data persiste (esto depende de la configuración de RDB)
kubectl port-forward -n proyecto3 svc/valkey-svc 6379:6379
redis-cli -h localhost -p 6379 DBSIZE
# Esperado: DBSIZE > 0 (data recuperada desde disco)
```

---

## Fase 9: Troubleshooting

### Problema: Pod en CrashLoopBackOff

```bash
# Ver logs del pod
kubectl logs -n proyecto3 <pod-name> --previous

# Describir pod para ver eventos
kubectl describe pod -n proyecto3 <pod-name>

# Checks:
# - ¿Imagen existe en Zot?
# - ¿imagePullSecrets está definido?
# - ¿Variables de entorno correctas?
```

### Problema: Consumer no consume mensajes

```bash
# Verificar que Consumer está corriendo
kubectl get pods -n proyecto3 -l app=consumer

# Ver logs
kubectl logs -n proyecto3 -l app=consumer

# Verificar conexión a RabbitMQ
kubectl exec -it -n proyecto3 -l app=consumer -- \
  nc -zv rabbitmq 5672

# Verificar que la queue existe
kubectl exec -it -n proyecto3 -l app=rabbitmq -- \
  rabbitmq-diagnostics queues | grep reports.usa
```

### Problema: Grafana no se conecta a Valkey

```bash
# Verificar datasource en Grafana
kubectl port-forward -n proyecto3 svc/grafana-svc 3000:3000
# http://localhost:3000 → Configuration → Data Sources → Valkey
# Check connectivity

# Verificar desde Grafana pod
kubectl exec -it -n proyecto3 -l app=grafana -- \
  nc -zv valkey-svc.proyecto3.svc.cluster.local 6379

# Ver logs de Grafana
kubectl exec -it -n proyecto3 -l app=grafana -- \
  tail -50 /var/log/grafana/grafana.log
```

### Problema: Gateway API no expone servicio

```bash
# Verificar Gateway
kubectl get gateway -n proyecto3
kubectl describe gateway -n proyecto3

# Verificar HTTPRoute
kubectl describe httproute -n proyecto3

# Verificar que la ruta es /grpc-201901271
kubectl get httproute -n proyecto3 -o json | jq '.items[0].spec.rules'

# Intentar conectar directamente al servicio Rust (sin Gateway)
kubectl port-forward -n proyecto3 svc/rust-api 3000:3000
curl http://localhost:3000/health
```

---

## Fase 10: Recolectar Evidencias

### 10.1 Screenshots para Documentación

```bash
# 1. Dashboard de Grafana con métricas
kubectl port-forward -n proyecto3 svc/grafana-svc 3000:3000
# Abrir http://localhost:3000 en navegador
# Screenshot: Dashboard con gráficos de reports, planes, naves

# 2. Pods running
kubectl get pods -n proyecto3 -o wide
# Screenshot o salida: todos los pods en Running

# 3. HPA scaling
kubectl get hpa -n proyecto3
# Screenshot: HPA with desired > current (durante carga)

# 4. Resultados de Locust
# Screenshot: Estadísticas finales de carga

# 5. Zot registry con imágenes
curl -k -u admin:admin123 https://${ZOT_HOST}:5000/v2/_catalog
# Screenshot: JSON con todas las imágenes
```

### 10.2 Logs para Documentación

```bash
# Recolectar logs de todos los componentes
mkdir -p logs

# Rust API
kubectl logs -n proyecto3 -l app=rust-api > logs/rust-api.log

# Go API Client
kubectl logs -n proyecto3 -l app=api-client > logs/api-client.log

# Go gRPC Server
kubectl logs -n proyecto3 -l app=grpc-server > logs/grpc-server.log

# Consumer
kubectl logs -n proyecto3 -l app=consumer > logs/consumer.log

# RabbitMQ
kubectl logs -n proyecto3 -l app=rabbitmq > logs/rabbitmq.log

# Crear tarball para entrega
tar czf deployment-logs-$(date +%Y%m%d).tar.gz logs/
```

---

## Conclusión

Si todos los verificaciones en las 10 fases pasaron exitosamente:

✓ Cluster GKE funcionando  
✓ RabbitMQ con persistencia  
✓ Rust API exponiendo /grpc-201901271  
✓ Go Services con 2 contenedores  
✓ KubeVirt VMs (Valkey + Grafana)  
✓ Consumer integrando RabbitMQ ↔ Valkey  
✓ Gateway API ruteando correctamente  
✓ HPA escalando bajo carga  
✓ Locust generando tráfico exitoso  
✓ Zot HTTPS registry con todas las imágenes  

**Proyecto3 M.U.M.N.K8s está listo para demostración y evaluación.**
