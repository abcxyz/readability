# Readability

This repository contains per-language readability requirements and also serves
as the source of truth for current readability members.

## Standards

Language-specific standards are available in the
[WiKi](https://github.com/abcxyz/readability/wiki).

## Proposing new readability

To propose new readability:

1. Find the associated language YAML file in the `terraform` directory. For
    example, to Go readability file is in `terraform/go.yaml`.

1. Propose a new membership under the members section:

    ```yaml
    github_username: 'member'
    ```

    for example:

    ```yaml
    sethvargo: 'member'
    ```

1. Create a Pull Request - GitHub will automatically assign the appropriate
    reviewers.

    If the Pull Request reports errors, inspect the output and fix errors.
    Usually these errors are due to improper formatting or invalid YAML.

## Enforcing readability on a project

By default, readabilty is not enforced. To enforce readability:

1. Grant `@abcxyz/<lang>-readability` `triage` or greater permissions on your
    repository.

1. Add the following line(s) to your [`CODEOWNERS`][codeowners] file:

    ```text
    *.ext    @abcxyz/<lang>-readability
    ```

    for example, for Go readability:

    ```text
    *.go    @abcxyz/go-readability
    ```

    if your language has multiple file extensions, add them all:

    ```text
    *.java     @abcxyz/java-readability
    *.scala    @abcxyz/java-readability
    ```

1. Ensure you have [branch protection rules][branch-protection-rules] to
    require a minimum number of reviewers. Specifically "Require review from
    Code Owners".

## Adding a new language

There are two types of languages used by `abcxyz`. Languages with broad adoption that are a standard part of our technology stack (i.e. Go, Terraform, etc.) and languages with limited use or that are in early adoption (i.e. Python, Dart, etc.).

### Limited use / early adoption languages

For new languages that we are uncertain about their widespread adoption, we will add targeted individuals as the CODEOWNERS for new repos. We will only convert them to a group and establish the standard readability practice after the usage of that language exceeds 5 repositories or 50k lines of code.

When a language exceeds the limited use phase a group will be created for it as is documented below.

### Add a new language for widespread adoption

1. Manually create a team in the organization.

    - **Name**: `<lang>-readability` (e.g. `go-readability`), all lowercase.
    - **Description**: `<lang> readability` (e.g. `Go readability`), normal case.
    - **Parent team**: `readability`.

1. Go to the team's settings page and click on "Code review" in the sidebar.

    - **Only notify requested team members**: Checked.
    - **Enable auto assignment**: Checked.
    - **How many team members should be assigned to review?**: 1.
    - **Routing algorithm**: Load balance.
    - **Team review assignment**: Uncheck.

    Leave all other options as the default.

1. (Optional) On the team's settings page, upload a logo.

1. Update `terraform/main.tf` and add a new readabilty stanza. The easiest
    thing to do is to copy an existing readability stanza and update `<lang>`.

1. Create a new YAML file `terraform/<lang>.yaml` and add at least one
    maintainer. If you are the language owner, you can add yourself.

1. Propose a Pull Request with the changes.

[codeowners]: https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners
[branch-protection-rules]: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/defining-the-mergeability-of-pull-requests/managing-a-branch-protection-rule
