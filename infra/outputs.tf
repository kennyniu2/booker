output "public_ip" {
  description = "Value for the DEPLOY_HOST GitHub secret"
  value       = aws_eip.booker.public_ip
}

output "ssh_command" {
  value = "ssh ec2-user@${aws_eip.booker.public_ip}"
}

output "backup_bucket" {
  value = aws_s3_bucket.backups.id
}
