# kubefin — Reseña del proyecto

Una CLI en Go que unifica el **diagnóstico de EKS** y el **análisis de costos AWS (FinOps)**,
con tres superficies sobre la misma lógica: **CLI**, **servidor MCP** y **agente de IA (Groq)**.
Reemplaza a `EKS-resources-review` (Python) y `scripts-sh`. Stack 100% gratis y read-only en v1.

## Ubicación de la documentación

Toda la documentación vive en:

```
/Users/jaime.henao/arheanja/scripts-hub/Scripts-Devops/kubefin/
├── README.md                 # overview, stack gratis, quickstart
├── docs/
│   ├── requirements.md       # requisitos EARS (R1–R7)
│   ├── design.md             # arquitectura y decisiones D1–D6
│   └── tasks.md              # plan por fases 0–5
└── docs/RESENA.md            # este documento
```

Rutas relativas desde la raíz del repo: [`README.md`](../README.md) ·
[`docs/requirements.md`](requirements.md) · [`docs/design.md`](design.md) ·
[`docs/tasks.md`](tasks.md)

## En una frase

De pod a factura: correlaciona el uso real de recursos del clúster con las recomendaciones
de rightsizing y el costo real, para optimizar el gasto — todo sobre tiers gratuitos.

## Puntos clave

- **Gratis por defecto:** datos de costo vía CUR en S3; la Cost Explorer API ($0.01/req) es opt-in con aviso.
- **IA con Groq** (free tier), proveedor intercambiable, salida estructurada.
- **Read-only v1:** nada que cree/modifique recursos AWS sin confirmación humana.
- **Agent-native:** cada capacidad es una tool reutilizable desde CLI, MCP y agente.

## Estado

Fase 0 (scaffold) en curso. El binario compila y responde a `--version` y `--help`.
Ver [`docs/tasks.md`](tasks.md) para el progreso por fases.
