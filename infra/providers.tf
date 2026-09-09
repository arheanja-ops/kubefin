terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Backend S3 opcional; configúralo con `terraform init -backend-config=...`.
  # backend "s3" {}
}

# El CUR (legacy) solo puede definirse en us-east-1, independientemente de la
# región donde vivan tus recursos.
provider "aws" {
  region = "us-east-1"
  alias  = "cur"
}

provider "aws" {
  region = var.region
}
