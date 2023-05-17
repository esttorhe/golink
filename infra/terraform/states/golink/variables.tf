variable "service_name" {
  default = "golink"
}
variable "aws_region" {
  default = "eu-central-1"
}

variable "aws_account_name" {
  default = "dev"
}
variable "cluster_name" {
  default = "preprod"
}

variable "k8s_namespace" {
  default     = "tooling"
  description = "the k8s namespace"
}
variable "container_port" {
  default = 80
}
variable "image_tag" {
  default = "latest"
}

variable "db_name" {
  default = "golinks"
}

variable "db_user" {
  default = "golink-admin"
}

variable "max_retries" {
  default = 5
}





