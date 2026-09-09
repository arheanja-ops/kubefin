variable "region" {
  description = "Región AWS para los recursos (el CUR export vive en us-east-1)."
  type        = string
  default     = "us-east-1"
}

variable "bucket_name" {
  description = "Nombre del bucket S3 donde se exporta el CUR. Debe ser único global."
  type        = string
}

variable "report_prefix" {
  description = "Prefijo (carpeta) dentro del bucket para el export del CUR."
  type        = string
  default     = "cur"
}

variable "report_name" {
  description = "Nombre del Cost & Usage Report."
  type        = string
  default     = "kubefin-cur"
}

variable "reader_principal_arns" {
  description = "ARNs de principals (usuarios/roles) autorizados a asumir el rol read-only del CUR."
  type        = list(string)
  default     = []
}

variable "tags" {
  description = "Tags aplicados a los recursos."
  type        = map(string)
  default = {
    Project   = "kubefin"
    ManagedBy = "terraform"
  }
}
