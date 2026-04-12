variable "region" {
  description = "AWS region"
  type        = string
  default     = "eu-west-2"
}

variable "app_name" {
  description = "Name used for all resources"
  type        = string
  default     = "sme-prototype"
}

variable "anthropic_api_key" {
  description = "Anthropic API key passed to the container as an environment variable"
  type        = string
  sensitive   = true
}

variable "auth_username" {
  description = "Login username for the app"
  type        = string
}

variable "auth_password" {
  description = "Login password for the app"
  type        = string
  sensitive   = true
}
