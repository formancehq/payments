resource "aws_secretsmanager_secret" "service" {
  name = "service/payments"
}

resource "aws_secretsmanager_secret_version" "service" {
  secret_id = aws_secretsmanager_secret.service.id
  secret_string = jsonencode({
    NUMARY_MONGODB_CONN_STRING = "mongodb+srv://${var.app_env_name}:${random_password.password.result}@${local.mongodb_atlas_url}"
  })
}

locals {
  mongodb_atlas_url = trimprefix(mongodbatlas_advanced_cluster.test.connection_strings.0.private_srv, "mongodb+srv://")
}


resource "mongodbatlas_advanced_cluster" "test" {
  project_id             = var.atlas_project_id
  name                   = var.app_env_name
  cluster_type           = "REPLICASET"
  backup_enabled         = true
  mongo_db_major_version = "5.0"
  replication_specs {
    region_configs {
      electable_specs {
        instance_size = "M10"
        node_count    = 3
      }
      provider_name = "AWS"
      priority      = 7
      region_name   = "EU_WEST_1"
    }
  }
}

resource "random_password" "password" {
  length           = 16
  special          = true
  override_special = "_%@"
}

resource "mongodbatlas_database_user" "user" {
  username           = var.app_env_name
  password           = random_password.password.result
  project_id         = var.atlas_project_id
  auth_database_name = "admin"

  roles {
    role_name     = "readWrite"
    database_name = "payments"
  }

  scopes {
    type = "CLUSTER"
    name = mongodbatlas_advanced_cluster.test.name
  }
}