# Plan de Implementación — kubefin

Cada tarea referencia los requisitos que satisface (ver [`requirements.md`](requirements.md)) y asume el diseño de [`design.md`](design.md). El orden respeta las dependencias: el core antes que las superficies, y las tools antes que el agente.

---

## Fase 0 — Higiene y scaffold

- [ ] 0.1 Inicializar módulo Go (`go mod init`), `.gitignore` (Go, `__pycache__`, artefactos de salida), y `git init`.
- [ ] 0.2 Crear el esqueleto de paquetes de `design.md` (`cmd/`, `internal/{cli,mcp,agent,ai,tools,model,render,awsx,k8sx}`).
- [ ] 0.3 Añadir `Makefile` (build/test/lint/run) y `golangci-lint` configurado.
- [ ] 0.4 Congelar `mac-optimization-tools` (nota en su README indicando que está fuera de alcance).
- [ ] 0.5 Wire mínimo de `cobra` en `cmd/kubefin/main.go` con `--version` y `--help`.
  - _Requisitos: R7.4, R7.5_

## Fase 1 — Core EKS (client-go)

- [ ] 1.1 Implementar `k8sx` (carga de kubeconfig, construcción de clientset y metrics client). Error claro si no hay contexto.
  - _Requisitos: R1.1, R1.2_
- [ ] 1.2 Definir structs en `model/` (Node, Pod, NamespaceUsage, Finding, etc.).
- [ ] 1.3 Implementar tools en `internal/tools/eks/`: estado de nodos, uso por nodo, top consumidores, recursos por namespace, quotas, reinicios de pods, requests/limits vs uso.
  - _Requisitos: R1.3_
- [ ] 1.4 Implementar tools Karpenter (config + errores), Datadog Agent coverage, componentes core EKS, recomendaciones.
  - _Requisitos: R1.3_
- [ ] 1.5 Manejo de Metrics Server ausente sin abortar el resto.
  - _Requisitos: R1.4_
- [ ] 1.6 `render`: tabla, JSON, Markdown, Confluence.
  - _Requisitos: R1.5_
- [ ] 1.7 Comandos CLI `kubefin eks ...` y `kubefin eks report`.
- [ ] 1.8 Tests de tools con `fake.NewSimpleClientset`.
  - _Requisitos: R7.1, R7.3_

## Fase 2 — Core FinOps gratuito

- [ ] 2.1 Implementar `awsx` (carga de config AWS, verificación de identidad, construcción de clientes S3/Compute Optimizer).
- [ ] 2.2 `infra/` en Terraform: bucket S3, CUR export, rol IAM read-only. Documentar que `apply` crea recursos (confirmación humana).
  - _Requisitos: R6.3, R7.6_
- [ ] 2.3 Implementar `tools/cost` — lector de CUR desde S3 (Parquet/CSV) agrupable por servicio/tag/periodo.
  - _Requisitos: R2.1, R2.4_
- [ ] 2.4 Mensaje guía cuando el CUR no está disponible (cuenta nueva).
  - _Requisitos: R2.5_
- [ ] 2.5 Ruta opt-in de CE API tras `--use-ce-api` con aviso de costo $0.01/req; rechazo si falta el flag.
  - _Requisitos: R2.2, R2.3, R6.2_
- [ ] 2.6 `tools/optimize` — Compute Optimizer (EC2/EBS) y correlación con requests-vs-uso de Fase 1; estimación de ahorro si hay CUR.
  - _Requisitos: R3.1, R3.2, R3.3, R3.4_
- [ ] 2.7 Comandos CLI `kubefin cost ...` y `kubefin optimize ...`.
- [ ] 2.8 Tests con AWS SDK mockeado y fixtures pequeños de CUR.
  - _Requisitos: R7.1, R7.3_

## Fase 3 — Superficies MCP y agente Groq

- [ ] 3.1 `internal/ai`: interfaz `Provider` + implementación Groq (API OpenAI-compatible), lectura de `GROQ_API_KEY`, manejo de 429 con backoff.
  - _Requisitos: R5.2, R5.6, R6.4_
- [ ] 3.2 Redacción opcional de datos sensibles (account IDs, ARNs, IPs) antes de envío remoto.
  - _Requisitos: R5.5_
- [ ] 3.3 `internal/mcp`: servidor MCP por stdio (go-sdk) que registra las tools core con esquema I/O; misma política de costo opt-in.
  - _Requisitos: R4.1, R4.2, R4.3, R4.4_
- [ ] 3.4 `internal/agent`: loop de tool-use con Groq que invoca tools iterativamente y devuelve `[]Finding` estructurado; nunca ejecuta escrituras.
  - _Requisitos: R5.1, R5.3, R5.4, R5.7 (D5)_
- [ ] 3.5 Comandos `kubefin mcp` y `kubefin ask "<pregunta>"`.
- [ ] 3.6 Tests con Provider mock (secuencias de tool-use deterministas).
  - _Requisitos: R7.1, R7.3_

## Fase 4 — CI/CD y distribución

- [ ] 4.1 GitHub Actions: `build → test → golangci-lint` en cada push, zero warnings.
  - _Requisitos: R7.5_
- [ ] 4.2 `.goreleaser.yaml`: binarios linux/darwin arm64+amd64 + GitHub Releases (opcional Homebrew tap).
  - _Requisitos: R7.4_
- [ ] 4.3 `Dockerfile` multi-stage para el servidor MCP.
- [ ] 4.4 Publicar repo en GitHub y documentar instalación.
- [ ] 4.5 (Opcional) GitOps: manifiesto del MCP server + Argo CD/Flux para despliegue en clúster.

## Fase 5 — Puente pod ↔ factura y retiro de lo viejo

- [ ] 5.1 Comando integrado que correlaciona rightsizing de pods (EKS) + Compute Optimizer + costo real del CUR en una recomendación FinOps con ahorro estimado.
  - _Requisitos: R3.2, R3.3_
- [ ] 5.2 Portar cualquier funcionalidad faltante de `scripts-sh` (karpenter.sh, cpu_usage.sh) como flags/subcomandos.
- [ ] 5.3 Verificar paridad con `EKS-resources-review` y **eliminar** `EKS-resources-review/` y `scripts-sh/`.

---

## Prerrequisitos externos (no son tareas de código)

- Renovar credenciales AWS: actualmente dan `InvalidClientTokenId`. Necesario para validación end-to-end (Fases 2+).
- Obtener `GROQ_API_KEY` en console.groq.com (gratis, sin tarjeta). Necesario para Fase 3.
- El CUR tarda ~24h en poblarse tras `terraform apply` (Fase 2).

## Estrategia de ejecución con sub-agentes

- Fases 1 y 2 son paralelizables (EKS vs FinOps, dominios independientes) → dos pipelines de research→implement.
- Fase 3 depende de que las tools de 1 y 2 existan → converge ambos pipelines.
- Un stage auditor independiente revisa tests y lint con loop-back en fallo.
