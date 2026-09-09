output "cur_bucket" {
  description = "Bucket S3 donde se exporta el CUR (usar con `kubefin cost --bucket`)."
  value       = aws_s3_bucket.cur.id
}

output "cur_prefix" {
  description = "Prefijo del export del CUR (usar con `kubefin cost --prefix`)."
  value       = var.report_prefix
}

output "reader_role_arn" {
  description = "ARN del rol IAM read-only para kubefin."
  value       = aws_iam_role.kubefin_reader.arn
}
