terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region

  default_tags {
    tags = { Project = "booker" }
  }
}

data "aws_caller_identity" "current" {}

# Latest Amazon Linux 2023 AMI for the chosen architecture (Graviton by default)
data "aws_ssm_parameter" "al2023" {
  name = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-${var.instance_arch}"
}

data "aws_vpc" "default" {
  default = true
}

# ---------- Networking ----------

resource "aws_security_group" "booker" {
  name_prefix = "booker-"
  description = "SSH only; the bot only makes outbound connections"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.ssh_cidrs
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_key_pair" "deploy" {
  key_name_prefix = "booker-deploy-"
  public_key      = var.ssh_public_key
}

# ---------- Backups bucket ----------

resource "aws_s3_bucket" "backups" {
  bucket = "booker-backups-${data.aws_caller_identity.current.account_id}-${var.region}"
}

resource "aws_s3_bucket_public_access_block" "backups" {
  bucket                  = aws_s3_bucket.backups.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "backups" {
  bucket = aws_s3_bucket.backups.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "backups" {
  bucket = aws_s3_bucket.backups.id
  rule {
    id     = "expire-old-backups"
    status = "Enabled"
    filter {}
    expiration {
      days = var.backup_retention_days
    }
  }
}

# ---------- Instance role (can only write to the backups bucket) ----------

data "aws_iam_policy_document" "assume_ec2" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "booker" {
  name_prefix        = "booker-"
  assume_role_policy = data.aws_iam_policy_document.assume_ec2.json
}

data "aws_iam_policy_document" "backups" {
  statement {
    actions   = ["s3:PutObject"]
    resources = ["${aws_s3_bucket.backups.arn}/*"]
  }
}

resource "aws_iam_role_policy" "backups" {
  name   = "write-backups"
  role   = aws_iam_role.booker.id
  policy = data.aws_iam_policy_document.backups.json
}

resource "aws_iam_instance_profile" "booker" {
  name_prefix = "booker-"
  role        = aws_iam_role.booker.name
}

# ---------- Instance ----------

resource "aws_instance" "booker" {
  ami                    = data.aws_ssm_parameter.al2023.value
  instance_type          = var.instance_type
  key_name               = aws_key_pair.deploy.key_name
  vpc_security_group_ids = [aws_security_group.booker.id]
  iam_instance_profile   = aws_iam_instance_profile.booker.name

  user_data = templatefile("${path.module}/user_data.sh.tpl", {
    db_user          = var.db_user
    db_password      = var.db_password
    db_name          = var.db_name
    discord_token    = var.discord_token
    discord_guild_id = var.discord_guild_id
    backup_bucket    = aws_s3_bucket.backups.id
  })

  metadata_options {
    http_tokens = "required" # IMDSv2 only
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = var.disk_size_gb
    encrypted   = true
  }

  tags = { Name = "booker" }

  lifecycle {
    # Don't replace the instance (and wipe the DB volume) when the AMI updates or secrets change.
    ignore_changes = [ami, user_data]
  }
}

# Stable address so the DEPLOY_HOST GitHub secret never changes
resource "aws_eip" "booker" {
  instance = aws_instance.booker.id
  domain   = "vpc"
}

# ---------- Cost guard ----------

resource "aws_budgets_budget" "monthly" {
  count        = var.budget_email == "" ? 0 : 1
  name         = "booker-monthly"
  budget_type  = "COST"
  limit_amount = tostring(var.monthly_budget_usd)
  limit_unit   = "USD"
  time_unit    = "MONTHLY"

  notification {
    comparison_operator        = "GREATER_THAN"
    threshold                  = 80
    threshold_type             = "PERCENTAGE"
    notification_type          = "ACTUAL"
    subscriber_email_addresses = [var.budget_email]
  }
}
