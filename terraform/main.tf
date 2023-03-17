# Main
terraform {
  required_version = "~> 1.3"

  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 5.15"
    }
  }
}

provider "github" {
  owner = "abcxyz"
}

# Go
data "github_team" "go_readability" {
  slug = "go-readability"
}

resource "github_team_members" "go_readability" {
  team_id = data.github_team.go_readability.id

  dynamic "members" {
    for_each = yamldecode(file("go.yaml"))

    content {
      username = members.key
      role     = members.value
    }
  }
}

# Java
data "github_team" "java_readability" {
  slug = "java-readability"
}

resource "github_team_members" "java_readability" {
  team_id = data.github_team.java_readability.id

  dynamic "members" {
    for_each = yamldecode(file("java.yaml"))

    content {
      username = members.key
      role     = members.value
    }
  }
}

# Typescript
data "github_team" "typescript_readability" {
  slug = "typescript-readability"
}

resource "github_team_members" "typescript_readability" {
  team_id = data.github_team.typescript_readability.id

  dynamic "members" {
    for_each = yamldecode(file("typescript.yaml"))

    content {
      username = members.key
      role     = members.value
    }
  }
}

# Terraform
data "github_team" "terraform_readability" {
  slug = "terraform-readability"
}

resource "github_team_members" "terraform_readability" {
  team_id = data.github_team.terraform_readability.id

  dynamic "members" {
    for_each = yamldecode(file("terraform.yaml"))

    content {
      username = members.key
      role     = members.value
    }
  }
}
