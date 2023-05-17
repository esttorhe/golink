provider "aws" {
  region              = var.aws_region
  max_retries         = var.max_retries
  allowed_account_ids = [module.init.aws["allowed_account_id"]]
  profile             = module.init.aws["profile"]
}
provider "kubernetes" {
  host     = module.init.k8s_config["host"]
  insecure = module.init.k8s_config["insecure"]
  exec {
    api_version = module.init.k8s_config["api_version"]
    command     = module.init.k8s_config["command"]
    args        = module.init.k8s_config["args"]
    env         = module.init.k8s_config["env"]
  }
}

provider "k8s" {

  host     = module.init.k8s_config["host"]
  insecure = module.init.k8s_config["insecure"]
  exec {
    api_version = module.init.k8s_config["api_version"]
    command     = module.init.k8s_config["command"]
    args        = module.init.k8s_config["args"]
    env         = module.init.k8s_config["env"]
  }
}

provider "postgresql" {
  host            = module.postgres.db_endpoint
  database        = var.db_name
  username        = var.db_user
  superuser       = false
  password        = module.postgres.db_auth
  connect_timeout = 15
  max_connections = 50
}

provider "cloudflare" {
  email   = module.init.cloudflare["email"]
  api_key = module.init.cloudflare["token"]
}
