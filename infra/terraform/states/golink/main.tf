terraform {
  backend "s3" {
    dynamodb_table = "terraform_state_lock"
    profile        = "terraform"
  }
}

module "k8s_init" {
  source = "git@github.com:zenjob/infra-code.git//terraform/modules/backend_init/v3"

  aws_region       = var.aws_region
  aws_account_name = var.aws_account_name
  k8s_cluster      = var.cluster_name

}

module "service" {
  source = "git@github.com:zenjob/infra-code.git//terraform/modules/backend_setup/v3"

  aws_region             = var.aws_region
  aws_account_name       = var.aws_account_name
  k8s_cluster            = var.k8s_cluster
  k8s_namespace          = var.k8s_namespace
  service_name           = var.service_name
  container_port         = var.container_port
  deployment_annotations = var.deployment_annotations
  template_labels        = var.extra_labels
  image_tag              = var.image_tag
  resources              = var.resources
  max_unavailability     = var.max_unavailability
  max_extended_capacity  = var.max_extended_capacity
  deployment_timeouts    = "20m"
  public_access          = true
  iam_role               = true

}
