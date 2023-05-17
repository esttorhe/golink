terraform {
  backend "s3" {
    dynamodb_table = "terraform_state_lock"
    profile        = "terraform"
  }
}

module "init" {
  source = "git@github.com:zenjob/infra-code.git//terraform/modules/backend_init/v3"

  aws_region       = var.aws_region
  aws_account_name = var.aws_account_name
  k8s_cluster      = var.cluster_name

}

module "service" {
  source = "git@github.com:zenjob/infra-code.git//terraform/modules/backend_setup/v3"

  aws_region          = var.aws_region
  aws_account_name    = var.aws_account_name
  k8s_cluster         = var.cluster_name
  k8s_namespace       = var.k8s_namespace
  service_name        = var.service_name
  container_port      = var.container_port
  image_tag           = var.image_tag
  deployment_timeouts = "20m"
  public_access       = false
  iam_role            = true
  depends = [
    module.postgres.depends,
    module.secrets.depends,
    module.config.depends
  ]

}

module "postgres" {
  source           = "git@github.com:zenjob/infra-code.git//terraform/modules/db_setup/v4"
  aws_region       = var.aws_region
  aws_account_name = var.aws_account_name
  service_name     = var.service_name
  k8s_namespace    = var.k8s_namespace
  k8s_cluster      = var.cluster_name
  db_name          = var.db_name
  db_user          = var.db_user
}

module "secrets" {
  source        = "git@github.com:zenjob/infra-code.git//terraform/modules/secret_setup/v4"
  k8s_namespace = var.k8s_namespace
  service_name  = var.service_name
  subkeys = {
    "DB_HOSTNAME" = "${module.postgres.db_endpoint}"
    "DB_USERNAME" = "${var.db_user}"
    "DB_PASSWORD" = "${module.postgres.db_auth}"
    "DB_PORT"     = 5432

  }

}
