# Upcoming CLI Automation Features

The following features are planned for the next release to provide full headless automation support for CI/CD pipelines. They are currently mock flags in the CLI.

## 1. `--export`
Fully bypass the TUI and export a project's SonarQube issues directly to the specified file path. 
*   Requires `--add-project` or pre-existing config to identify the project.
*   Requires the SonarQube token to be available via `USER_TOKEN` in the environment.

## 2. `--dry-run`
Fetch issues from the SonarQube API and print a summary to the console without saving the data to a CSV file.

## 3. `--quiet` / `-q`
Run the entire operation headlessly. No TUI, no spinner, only standard output or JSON-formatted logs suitable for machine parsing.
