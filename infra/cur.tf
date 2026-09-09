# --- Cost & Usage Report (CUR) ---
# Debe crearse en us-east-1 (proveedor con alias "cur").

resource "aws_cur_report_definition" "kubefin" {
  provider = aws.cur

  report_name                = var.report_name
  time_unit                  = "DAILY"
  format                     = "textORcsv"
  compression                = "GZIP"
  additional_schema_elements = ["RESOURCES"]

  s3_bucket = aws_s3_bucket.cur.id
  s3_prefix = var.report_prefix
  s3_region = "us-east-1"

  additional_artifacts   = []
  refresh_closed_reports = true
  report_versioning      = "OVERWRITE_REPORT"

  depends_on = [aws_s3_bucket_policy.cur]
}
