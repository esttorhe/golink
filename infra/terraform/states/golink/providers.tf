provider "aws" {
  region              = var.aws_region
  max_retries         = var.max_retries
  allowed_account_ids = [module.k8s_init.aws["allowed_account_id"]]
  profile             = module.k8s_init.aws["profile"]
}
provider "k8s" {
  host     = module.k8s_init.k8s_config["host"]
  insecure = module.k8s_init.k8s_config["insecure"]
  exec {
    api_version = module.k8s_init.k8s_config["api_version"]
    command     = module.k8s_init.k8s_config["command"]
    args        = module.k8s_init.k8s_config["args"]
    env         = module.k8s_init.k8s_config["env"]
  }
}
provider "kubernetes" {
  host     = module.k8s_init.k8s_config["host"]
  insecure = module.k8s_init.k8s_config["insecure"]
  exec {
    api_version = module.k8s_init.k8s_config["api_version"]
    command     = module.k8s_init.k8s_config["command"]
    args        = module.k8s_init.k8s_config["args"]
    env         = module.k8s_init.k8s_config["env"]
  }
}
