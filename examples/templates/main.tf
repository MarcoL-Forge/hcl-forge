terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

# Web server instance
resource "aws_instance" "web" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.micro"
  monitoring    = false

  tags = {
    Name        = "web-server"
    Environment = var.environment
  }
}

# Spot instance request (used when use_spot_instances = true)
resource "aws_spot_instance_request" "web_spot" {
  ami           = "ami-0c55b159cbfafe1f0"
  instance_type = "t3.micro"
  spot_price    = "0.03"

  tags = {
    Name        = "web-spot"
    Environment = var.environment
  }
}

# S3 bucket for static assets
resource "aws_s3_bucket" "assets" {
  bucket = "my-default-bucket"
  acl    = "private"

  tags = {
    Name        = "assets"
    Environment = var.environment
  }
}
