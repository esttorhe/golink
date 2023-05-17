variable "service_name" {
  default = "golink"
}
variable "aws_region" {
  default = "eu-central-1"
}

variable "aws_account_name" {
  default = "dev"
}
variable "k8s_cluster" {
  default = "dev"
}

variable "k8s_namespace" {
  default     = "tooling"
  description = "the k8s namespace"
}
variable "container_port" {
  default = 80
}


