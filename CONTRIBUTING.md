# Contributing to Heimdall

First off, thank you for considering contributing! Heimdall is an open-source project, and it thrives on community contributions. We're excited to have you here.

This document outlines the guidelines for contributing to Heimdall. Please read it carefully to ensure a smooth and effective contribution process.

: All participants are expected to follow our Code of Conduct.

## How Can I Contribute?

There are many ways to contribute, not just with code:

- **Reporting Bugs:** Find a bug? Please(https[://github.com/nexlified/heimdall/issues/new?template=01_bug_report.yml]) and provide as much detail as possible.
    
- **Suggesting Features:** Have an idea?(https[://github.com/nexlified/heimdall/issues/new?template=02_feature_request.yml]).
    
- **Improving Documentation:** Find a typo or a confusing section? Feel free to open a Pull Request.
    
- **Writing Code:** Help us fix bugs or build new features.
    

## Your First Code Contribution

Ready to write some code? Here is the standard workflow for getting your changes merged.

### 1. Fork the Repository

Click the "Fork" button at the top right of the . This will create a copy of the project in your own GitHub account.

### 2. Clone Your Fork

Clone your fork to your local machine:bash git clone cd heimdall

### 4. Create a New Branch

**Always** create a new branch for your changes. Name it something descriptive, like `fix/login-bug` or `feat/add-new-endpoint`.

### 5. Set Up Your Development Environment

We use Docker Compose to manage the development environment. This makes setup a one-click process.

Unlike the _adopter_ quickstart, you will use the **`docker-compose.dev.yml`** file. This file is configured to build the Heimdall services from your local source code and will automatically reload when you make changes.

Your local Heimdall source code (e.g., in `./heimdall-proxy`) is now mounted into the running container. Any changes you save will trigger a live-reload.

### 6. Make Your Changes

Write your code! Be sure to follow the project guidelines.

### 7. Commit and Push

Once your changes are ready, commit them with a clear message:

### 8. Open a Pull Request (PR)

Go to your fork on GitHub. You will see a prompt to "Compare & pull request". Click it.

- Set the **base repository** to `nexlified/heimdall` and the **base branch** to `main`.
    
- Set the **head repository** to `/heimdall` and the **compare branch** to `feat/my-amazing-feature`.
    
- Fill out the Pull Request template with details about your changes. Reference any issues your PR fixes (e.g., "Closes #42").
    

A maintainer will review your PR, provide feedback, and merge it once it's ready.

## Project Guidelines

### Go (Golang)

- We use **Go Modules**.
    
- We follow the standard [golang-standards/project-layout].
    
- We use [golangci-lint] for linting. Please run `golangci-lint run` before committing.
    

### API Changes

If your contribution changes an API:

- You **must** update the OpenAPI (Swagger) annotations in the Go code comments.
    
- You **must** update the `docs/API_REFERENCE.md` or other relevant documentation.
    

### Submitting a PR

- Please fill out the(.github/PULL_REQUEST_TEMPLATE.md) completely.
    
- Ensure the CI pipeline (GitHub Actions) passes for your PR.
    
- Do not force-push to a branch that has an open PR. If you need to make changes based on a review, add new commits.
