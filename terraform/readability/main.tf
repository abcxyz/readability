terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 6.5"
    }
  }
}

variable "language" {
  type        = string
  description = "Name of the language for readability."
}

locals {
  members = yamldecode(file("${var.language}.yaml"))
}

data "github_team" "readability" {
  slug = "${var.language}-readability"
}

resource "github_team_members" "readability" {
  team_id = data.github_team.readability.id

  dynamic "members" {
    for_each = local.members

    content {
      username = members.key
      role     = members.value
    }
  }
}

data "github_team" "readability_approvers" {
  slug = "${var.language}-readability-approvers"
}

resource "github_team_members" "readability_approvers" {
  team_id = data.github_team.readability_approvers.id

  dynamic "members" {
    for_each = {
      for key, val in local.members : key => val if val == "maintainer"
    }

    content {
      username = members.key
      role     = members.value
    }
  }
}
