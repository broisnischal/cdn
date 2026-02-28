terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

provider "aws" {
  alias  = "us"
  region = "us-east-1"
}

provider "aws" {
  alias  = "in"
  region = "ap-south-1"
}

provider "aws" {
  alias  = "eu"
  region = "eu-central-1"
}
