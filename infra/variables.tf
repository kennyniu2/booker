variable "region" {
  type    = string
  default = "us-east-1"
}

variable "instance_type" {
  type    = string
  default = "t4g.small"
}

variable "instance_arch" {
  description = "arm64 for Graviton (t4g) instance types, x86_64 otherwise"
  type        = string
  default     = "arm64"
}

variable "disk_size_gb" {
  type    = number
  default = 20
}

variable "ssh_cidrs" {
  description = "CIDRs allowed to SSH. GitHub Actions deploys over SSH too, see the README for the options."
  type        = list(string)
}

variable "ssh_public_key" {
  description = "Public key (contents of the .pub file) for the deploy keypair"
  type        = string
}

variable "db_user" {
  type    = string
  default = "booker"
}

variable "db_name" {
  type    = string
  default = "booker"
}

variable "db_password" {
  type      = string
  sensitive = true
}

variable "discord_token" {
  type      = string
  sensitive = true
}

variable "discord_guild_id" {
  description = "Optional: register slash commands to one guild (instant) instead of globally"
  type        = string
  default     = ""
}

variable "backup_retention_days" {
  type    = number
  default = 30
}

variable "budget_email" {
  description = "Email for an 80% monthly budget alert; empty disables the budget"
  type        = string
  default     = ""
}

variable "monthly_budget_usd" {
  type    = number
  default = 25
}
