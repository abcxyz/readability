terraform {
  required_version = "~> 1.3"

  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 5.42"
    }
  }
}

provider "github" {
  owner = "abcxyz"
}

module "readability" {
  for_each = toset([
    "go",
    "java",
    "terraform",
    "typescript",
  ])

  source   = "./readability"
  language = each.key
}
