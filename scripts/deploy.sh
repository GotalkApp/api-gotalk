#!/bin/bash
# ============================================================
# GoTalk API - Manual deploy script (backup khi không dùng CI/CD)
# Dùng để gọi thủ công trên VPS: bash scripts/deploy.sh anhnq996/gotalk-api:v260220.5
# ============================================================

set -e

IMAGE="${1:-anhnq996/gotalk-api:latest}"
NAMESPACE="${GOTALK_NAMESPACE:-gotalk}"
DEPLOYMENT="${GOTALK_DEPLOYMENT:-gotalk-api}"

echo "🚀 Deploying GoTalk API..."
echo "   Image:      ${IMAGE}"
echo "   Namespace:  ${NAMESPACE}"
echo "   Deployment: ${DEPLOYMENT}"
echo ""

# Kéo image mới nhất về node (optional, kubectl set image cũng tự làm)
# docker pull "${IMAGE}"

# Cập nhật image trên Kubernetes
kubectl set image deployment/${DEPLOYMENT} \
  api=${IMAGE} \
  --namespace=${NAMESPACE}

# Chờ rollout hoàn thành
echo "⏳ Waiting for rollout..."
kubectl rollout status deployment/${DEPLOYMENT} \
  --namespace=${NAMESPACE} \
  --timeout=180s

echo ""
echo "✅ Done! Current pods:"
kubectl get pods --namespace=${NAMESPACE} -l app=${DEPLOYMENT}
