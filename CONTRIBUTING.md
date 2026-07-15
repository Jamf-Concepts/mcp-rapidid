# How to contribute

We'd love to accept your patches and contributions to this project. There are
a just a few small guidelines you need to follow.

## Reporting issues

Bugs, feature requests, and development-related questions should be directed to
our [GitHub issue tracker](https://github.com/Jamf-Concepts/mcp-rapidid/issues). If
reporting a bug, please try and provide as much context as possible such as
your operating system, Go version, and anything else that might be relevant to
the bug. For feature requests, please explain what you're trying to do, and
how the requested feature would help you do that.

## Submitting a patch

1. It's generally best to start by opening a new issue describing the bug or
   feature you're intending to fix. Even if you think it's relatively minor,
   it's helpful to know what people are working on. It also provides the
   ability for others to provide feedback and guidance before work is started.
   Mention in the initial issue that you are planning to work on that bug or
   feature so that it can be assigned to you.

2. Maintainers review all issues and add the ready label once the approach
   for an issue has been discussed and confirmed. We recommend waiting for
   this label before starting work, so the scope and approach are settled
   before any code is written.

3. Follow the normal process of [forking][1] the project, and set up a new branch
   to work in. It's important that each group of changes be done in separate
   branches in order to ensure that a pull request only includes the commits
   related to that bug or feature.

4. Run `script/fmt.sh` to format your code

5. Do your best to have [well-formed commit messages][2] for each change. This
   provides consistency throughout the project, and ensures that commit messages
   are able to be formatted properly by various git tools.

6. Finally, push the commits to your fork and submit a [pull request][3].
   **NOTE:** Please do not use force-push on PRs in this repo, as it makes it
   more difficult for reviewers to see what has changed since the last code
   review. We always perform "squash and merge" actions on PRs in this repo, so it doesn't
   matter how many commits your PR has, as they will end up being a single commit after merging.
   This is done to make a much cleaner `git log` history and helps to find regressions in the code
   using existing tools such as `git bisect`.

[1]: https://help.github.com/articles/fork-a-repo
[2]: http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html
[3]: https://help.github.com/articles/creating-a-pull-request

## Scripts

The `script` directory has shell scripts that help with common development
tasks.

**script/fmt.sh** formats all go code in the repository.

## Other notes on code organization

Currently, all the tools are built in the internal ri package and then added
to the MCP server in the main.go file. Each RapidID tool should be in
its own file.

When releasing a new version ensure the version is updated in `manifest.json`
as this is the version that is displayed when installed using MCP Bundle.
