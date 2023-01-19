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
data "github_team" "go-readability" {
  slug = "go-readability"
}

resource "github_team_members" "go-readability" {
  team_id = data.github_team.go-readability.id

  dynamic "members" {
    for_each = yamldecode(file("go.yaml"))

    content {
      username = members.key
      role     = members.value
    }
  }
}

# Java
data "github_team" "java-readability" {
  slug = "java-readability"
}

resource "github_team_members" "java-readability" {
  team_id = data.github_team.java-readability.id

  dynamic "members" {
    for_each = yamldecode(file("java.yaml"))

    content {
      username = members.key
      role     = members.value
    }
  }
}

# Typescript
data "github_team" "typescript-readability" {
  slug = "typescript-readability"
}

resource "github_team_members" "typescript-readability" {
  team_id = data.github_team.typescript-readability.id

  dynamic "members" {
    for_each = yamldecode(file("typescript.yaml"))

    content {
      username = members.key
      role     = members.value
    }
  }
}
