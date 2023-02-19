# Contributing to Cleanenv

Thank you for your interest in contributing to Cleanenv!

## How to Contribute

Contributions can take many forms, including bug reports, feature requests, documentation improvements, and code changes. We welcome all contributions that align with the goals of the project.

Please keep in mind that this project is designed to be minimalistic, obvious, and as simple as possible. 
We prioritize clear and easy-to-understand code over complex or "magical" solutions. 
If you're considering making a code change that might be considered overcomplicated or not in line with the project's goals, 
we recommend opening an issue to discuss the change before submitting a pull request.

#### Note on environment variables names

Cleanenv strives to be as explicit and as obvious as possible, and as such, 
we decided to write environment variable in config struct tags *as is*.
It was done for purpose. This decision guarantees that programmer can *always* find environment variable by its name in code.
It may not sound pretty useful for such a limitation, but in my own experience, it is.
So you always will be able to find environment variable by its whole name in code,
which makes writing complex nested config structures a bit more difficult.

By the same reason, any contribution changing this behavior will be rejected.
Please not add any prefixes or suffixes to environment variables names, we will just not approve such PRs.
Thanks for understanding.

### Reporting Bugs and Requesting Features

If you've found a bug or have an idea for a new feature, please open an issue on our GitHub repository. Be sure to include as much detail as possible, including steps to reproduce the issue, expected behavior, and actual behavior.

### Contributing Code Changes

If you would like to contribute code changes to the project, please follow these steps:

1. Fork the repository to your own GitHub account.
2. Create a new branch for your changes.
3. Make your changes and test them thoroughly.
4. Submit a pull request (PR) with your changes.
5. Wait for a project maintainer to review your PR. The review process may take some time, so please be patient.
6. Respond to any feedback and make any necessary changes to your code.
7. Once your PR has been approved, it will be merged into the main branch of the repository.

### Code Style and Standards

Try to stick to the existing code style of the project.

### Testing

We rely on unit tests to ensure the quality of our code. 
Before submitting a PR, please make sure your changes include appropriate tests.
The number indicating the test code coverage should not decrease.

## Getting Help

If you need help with the project, please don't hesitate to ask. 
You can get in touch with the project maintainers by opening an issue on the GitHub repository.
