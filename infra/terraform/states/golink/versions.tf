terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }

    k8s = {
      source = "banzaicloud/k8s"
    }
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
    postgresql = {
      source = "cyrilgdn/postgresql"
    }

    cloudflare = {
      source = "cloudflare/cloudflare"
    }

  }
  required_version = ">= 1.0"
}
