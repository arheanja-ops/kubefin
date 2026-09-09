# --- Rol IAM read-only para kubefin (R7.6) ---
# Permisos mínimos: leer los objetos del CUR y las recomendaciones de Compute
# Optimizer. La CE API se incluye pero solo se invoca con --use-ce-api (opt-in).

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      type        = "AWS"
      identifiers = length(var.reader_principal_arns) > 0 ? var.reader_principal_arns : [data.aws_caller_identity.current.arn]
    }
  }
}

resource "aws_iam_role" "kubefin_reader" {
  name               = "kubefin-cur-reader"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
  tags               = var.tags
}

data "aws_iam_policy_document" "reader" {
  statement {
    sid       = "ReadCURBucket"
    effect    = "Allow"
    actions   = ["s3:GetObject", "s3:ListBucket"]
    resources = [aws_s3_bucket.cur.arn, "${aws_s3_bucket.cur.arn}/*"]
  }

  statement {
    sid    = "ReadComputeOptimizer"
    effect = "Allow"
    actions = [
      "compute-optimizer:GetEC2InstanceRecommendations",
      "compute-optimizer:GetEBSVolumeRecommendations",
      "compute-optimizer:GetEnrollmentStatus",
    ]
    resources = ["*"]
  }

  statement {
    sid       = "VerifyIdentity"
    effect    = "Allow"
    actions   = ["sts:GetCallerIdentity"]
    resources = ["*"]
  }

  # Cost Explorer API — solo se invoca con --use-ce-api (opt-in, con costo).
  statement {
    sid       = "ReadCostExplorer"
    effect    = "Allow"
    actions   = ["ce:GetCostAndUsage"]
    resources = ["*"]
  }
}

resource "aws_iam_role_policy" "reader" {
  name   = "kubefin-cur-reader-policy"
  role   = aws_iam_role.kubefin_reader.id
  policy = data.aws_iam_policy_document.reader.json
}
