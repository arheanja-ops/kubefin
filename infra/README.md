# infra/ — CUR export + IAM read-only (Terraform)

Provisiona la fuente de costo gratuita de kubefin:

- **Bucket S3** cifrado y privado para el Cost & Usage Report (CUR).
- **CUR report definition** (diario, CSV/GZIP, con recursos) en `us-east-1`.
- **Rol IAM read-only** (`kubefin-cur-reader`) con permisos mínimos: leer el CUR,
  Compute Optimizer, `sts:GetCallerIdentity` y `ce:GetCostAndUsage` (esta última
  solo se invoca con `--use-ce-api`).

> ⚠️ **`terraform apply` crea recursos en tu cuenta AWS.** El bucket S3 no tiene
> costo relevante (centavos de storage) y el CUR es gratis, pero cualquier
> creación de infraestructura requiere tu confirmación explícita. kubefin nunca
> ejecuta `apply` por ti.

## Uso

```bash
cd infra
terraform init
terraform plan  -var bucket_name=mi-cur-bucket-unico
terraform apply -var bucket_name=mi-cur-bucket-unico   # crea recursos AWS
```

Tras el `apply`, AWS tarda hasta **~24 h** en poblar el primer reporte. Luego:

```bash
kubefin cost by-service --bucket $(terraform output -raw cur_bucket) \
                        --prefix $(terraform output -raw cur_prefix) --last 30d
```

## Variables principales

| Variable | Descripción | Default |
|---|---|---|
| `bucket_name` | Nombre único global del bucket S3 | (requerido) |
| `report_prefix` | Prefijo del export | `cur` |
| `report_name` | Nombre del CUR | `kubefin-cur` |
| `reader_principal_arns` | ARNs que pueden asumir el rol reader | la identidad actual |
| `region` | Región de recursos (el CUR siempre en us-east-1) | `us-east-1` |
