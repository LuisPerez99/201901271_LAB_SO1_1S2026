#!/bin/bash

# Script: build-and-push-images.sh
# Propósito: Construir y publicar todas las imágenes Docker a Zot HTTPS registry
# Uso: ./build-and-push-images.sh <ZOT_REGISTRY_URL>
# Ejemplo: ./build-and-push-images.sh 35.192.12.45:5000

set -e

# Validar argumentos
if [ $# -ne 1 ]; then
    echo "Uso: $0 <ZOT_REGISTRY_URL>"
    echo "Ejemplo: $0 35.192.12.45:5000"
    exit 1
fi

ZOT_REGISTRY="$1"
CARNET="201901271"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "=========================================="
echo "M.U.M.N.K8s - Build and Push to Zot"
echo "=========================================="
echo "Registry: ${ZOT_REGISTRY}"
echo "Carnet: ${CARNET}"
echo "Project Root: ${PROJECT_ROOT}"
echo ""

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Función helper para print
print_step() {
    echo -e "${YELLOW}[STEP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Verificar que Docker está corriendo
print_step "Verificando Docker daemon..."
if ! docker ps &>/dev/null; then
    print_error "Docker daemon no está corriendo"
    exit 1
fi
print_success "Docker daemon OK"

# Verificar conectividad a Zot
print_step "Verificando conectividad a Zot..."
if ! curl -k -s -f "https://${ZOT_REGISTRY}/v2/" > /dev/null 2>&1; then
    print_error "No se puede alcanzar Zot en ${ZOT_REGISTRY}"
    echo "Intenta:"
    echo "  1. Verificar IP de VM: gcloud compute instances list"
    echo "  2. Verificar firewall: gcloud compute firewall-rules list"
    echo "  3. Verificar Zot corriendo: sudo systemctl status zot"
    exit 1
fi
print_success "Zot reachable"

# Array de servicios a construir
declare -a SERVICES=(
    "api:rust-api"
    "go-api-client:api-client"
    "go-grpc-server:grpc-server"
    "go-consumer:go-consumer"
)

# Función para construir y publicar una imagen
build_and_push() {
    local service_dir=$1
    local image_name=$2
    local full_image_tag="${ZOT_REGISTRY}/${image_name}:${CARNET}"
    
    print_step "Construyendo ${image_name}..."
    
    if [ ! -d "${PROJECT_ROOT}/${service_dir}" ]; then
        print_error "Directorio no encontrado: ${PROJECT_ROOT}/${service_dir}"
        return 1
    fi
    
    cd "${PROJECT_ROOT}/${service_dir}"
    
    # Construir imagen
    if docker build -t "${full_image_tag}" . &>/dev/null; then
        print_success "Build exitoso: ${full_image_tag}"
    else
        print_error "Build falló para ${image_name}"
        return 1
    fi
    
    # Publicar a Zot
    print_step "Publicando ${full_image_tag} a Zot..."
    
    if docker push "${full_image_tag}" &>/dev/null; then
        print_success "Push exitoso: ${full_image_tag}"
        echo "  → Verificar: curl -k -u admin:admin123 https://${ZOT_REGISTRY}/v2/${image_name}/tags/list"
    else
        print_error "Push falló para ${full_image_tag}"
        return 1
    fi
    
    return 0
}

# Contador de éxitos/fallos
TOTAL=0
SUCCESS=0
FAILED=0

# Construir y publicar cada servicio
for service_pair in "${SERVICES[@]}"; do
    IFS=':' read -r service_dir image_name <<< "$service_pair"
    TOTAL=$((TOTAL + 1))
    
    if build_and_push "$service_dir" "$image_name"; then
        SUCCESS=$((SUCCESS + 1))
    else
        FAILED=$((FAILED + 1))
    fi
    
    echo ""
done

# Resumen final
echo "=========================================="
echo "Resumen de Build & Push"
echo "=========================================="
echo "Total: ${TOTAL}"
echo -e "Success: ${GREEN}${SUCCESS}${NC}"
echo -e "Failed: ${RED}${FAILED}${NC}"
echo ""

# Verificación final
print_step "Verificando imágenes en Zot..."
echo ""

for service_pair in "${SERVICES[@]}"; do
    IFS=':' read -r service_dir image_name <<< "$service_pair"
    
    echo "Imagen: ${image_name}"
    if curl -k -s -u admin:admin123 "https://${ZOT_REGISTRY}/v2/${image_name}/tags/list" 2>/dev/null | grep -q "${CARNET}"; then
        echo -e "  ${GREEN}✓ Disponible${NC}"
    else
        echo -e "  ${RED}✗ No encontrada${NC}"
    fi
done

echo ""

if [ $FAILED -eq 0 ]; then
    print_success "Todas las imágenes construidas y publicadas correctamente"
    echo ""
    echo "Próximos pasos:"
    echo "1. Crear secret en K8s:"
    echo "   kubectl create secret docker-registry zot-credentials \\"
    echo "     --docker-server=${ZOT_REGISTRY} \\"
    echo "     --docker-username=admin \\"
    echo "     --docker-password=admin123 \\"
    echo "     -n proyecto3"
    echo ""
    echo "2. Actualizar manifests K8s con:"
    echo "   image: ${ZOT_REGISTRY}/rust-api:${CARNET}"
    echo "   imagePullPolicy: IfNotPresent"
    echo ""
    exit 0
else
    print_error "Algunos builds/pushes fallaron"
    exit 1
fi
