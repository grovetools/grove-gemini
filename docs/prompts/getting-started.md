# Getting Started Guide

You are writing a quick-start tutorial for a new user of `gemapi`.

## Task
Create a step-by-step guide that walks the user through their first few commands. The tutorial should cover:
1.  A basic `gemapi request` with a simple text prompt.
2.  Introducing a `.grove/rules` file to a project and running a request that automatically includes code context. Explain how `gemapi` uses this file.
3.  Demonstrating the caching feature. Explain that caching is opt-in and requires `@enable-cache` in the rules file. Show a `gemapi request` that creates a cache, and then a second one that uses the cache.
4.  Finishing with a simple observability command, like `gemapi query local`, to show the user how they can see their request history.

## Output Format
- A friendly, tutorial-style tone.
- Clear, numbered steps.
- Code blocks for file contents (e.g., `main.go`, `.grove/rules`) and shell commands.
- Explanations of the output the user should expect to see.