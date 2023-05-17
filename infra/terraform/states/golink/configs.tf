module "config" {
  source        = "git@github.com:zenjob/infra-code.git//terraform/modules/configs_setup/v1"
  k8s_namespace = var.k8s_namespace
  service_name  = var.service_name
  subkeys = {
    "dataSource.url"      = "jdbc:postgresql://${module.postgres.db_endpoint}:5432/${var.db_name}?ApplicationName=${var.service_name}"
    "dataSource.username" = "${var.db_user}"
    "dataSource.password" = "${module.postgres.db_auth}"
  }
}
