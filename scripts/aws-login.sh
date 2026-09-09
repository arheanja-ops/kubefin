#!/usr/bin/env bash
# Verifica la sesión SSO del profile personal (AWS_PROFILE del .envrc) y la renueva si expiró.
# kubefin usa este perfil para el análisis de costos (CUR en S3) en Fase 2 — read-only.
# Uso: ./scripts/aws-login.sh
set -euo pipefail

PROFILE="${AWS_PROFILE:-taxops-admin}"

echo "→ Verificando sesión AWS SSO (profile: ${PROFILE})..."

if aws sts get-caller-identity --profile "${PROFILE}" >/dev/null 2>&1; then
  echo "✅ Sesión activa:"
  aws sts get-caller-identity --profile "${PROFILE}" --output table
else
  echo "⏳ Sesión expirada o no iniciada — abriendo el navegador para reautorizar..."
  aws sso login --profile "${PROFILE}"
  echo "✅ Login completado:"
  aws sts get-caller-identity --profile "${PROFILE}" --output table
fi
