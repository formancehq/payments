terraform {
  backend "remote" {
    organization = "numary"

    workspaces {
      prefix = "app-payments-"
    }
  }
  required_providers {
    mongodbatlas = {
      source  = "mongodb/mongodbatlas"
      version = "1.2.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}

provider "mongodbatlas" {}

provider "aws" {
  region = local.region
  default_tags {
    tags = {
      Environment = var.env
      App         = local.app
    }
  }
}

locals {
  region = "eu-west-1"
  app    = "payments"
}

variable "env" {}
variable "vpc_id" {}
variable "vpc_cidr" {}
variable "subnet_a" {}
variable "subnet_b" {}
variable "subnet_c" {}
variable "app_env_name" {}
variable "atlas_project_id" {}
