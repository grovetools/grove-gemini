This guide provides a hands-on tutorial to get you started with the `gemapi` CLI. You will learn how to make basic requests, automatically include code context from your project, use the caching features to save costs, and review your usage history.

Before you begin, ensure you have installed `gemapi` and configured your Gemini API key. For detailed instructions, see the [Installation & Configuration](./installation-and-configuration.md) guide.

### Step 1: Your First Request

The simplest way to use `gemapi` is to provide a prompt directly on the command line. This sends your text to the Gemini API without any additional file context.

1.  Open your terminal and run the following command:

    ```bash
    gemapi request "What is the Go programming language?"
    ```

2.  You will see output indicating that `gemapi` is calling the API, followed by the generated response.

    ```text
    🏠 Working directory: /path/to/your/project
    ⚠️ No .grove/rules file found - context management disabled
    💡 Create .grove/rules to enable automatic context inclusion

    🤖 Calling Gemini API with model: gemini-2.0-flash


    🤖 Generating response...

    📊 Token usage:
    ╭────────────────────────────────╮
    │ Cold (Cached): 0 tokens        │
    │ Hot (Dynamic): 0 tokens        │
    │ User Prompt: 7 tokens          │
    │ -------------------------------- │
    │ Total Prompt: 7 tokens         │
    │ Completion: 123 tokens         │
    │ -------------------------------- │
    │ Total API Usage: 130 tokens    │
    │ Cache Hit Rate: 0.0%           │
    │ Response Time: 1.23s           │
    ╰────────────────────────────────╯

    The Go programming language, often referred to as Golang, is a statically typed, compiled programming language designed at Google...
    ```

This is the most basic usage. The tool informs you that context management is disabled because it didn't find a `.grove/rules` file.

### Step 2: Adding Code Context

`gemapi`'s primary function is to automatically include relevant code from your project as context for your requests. This is controlled by a `.grove/rules` file.

1.  First, let's create a small sample project.

    ```bash
    mkdir my-go-project
    cd my-go-project
    ```

2.  Create a Go file named `main.go`.

    ```go
    // File: main.go
    package main

    import "fmt"

    func main() {
        message := "Hello from gemapi!"
        fmt.Println(message)
    }
    ```

3.  Now, create the directory and rules file that `gemapi` uses.

    ```bash
    mkdir .grove
    echo "*.go" > .grove/rules
    ```
    This simple rule tells `gemapi` to include all files ending with `.go` in the context.

4.  Run a request that asks a question about the code.

    ```bash
    gemapi request "Explain the purpose of the main.go file in this project."
    ```

    The output will now show that `gemapi` has found and used your rules file to include `main.go` in the request.

    ```text
    🏠 Working directory: /path/to/my-go-project
    📋 Found rules file: /path/to/my-go-project/.grove/rules
    ...
    📁 Files attached to request:
      • /path/to/my-go-project/.grove/context
    ...
    🤖 Generating response...
    ...
    ```

The model's response will now be based on the content of `main.go`, providing a specific answer about your project.

### Step 3: Using the Cache

For large projects, sending the same code context with every request can be slow and costly. `gemapi` solves this by caching the "cold context" (large, infrequently changing files) using the Gemini Caching API. This feature is opt-in.

1.  Enable caching by adding the `@enable-cache` directive to your rules file.

    ```bash
    # You can use 'echo' to prepend the directive to the file
    echo -e "@enable-cache\n$(cat .grove/rules)" > .grove/rules
    ```

    Your `.grove/rules` file should now look like this:
    ```
    @enable-cache
    *.go
    ```

2.  Run the same request again. This time, `gemapi` will detect that caching is enabled and create a new cache.

    ```bash
    # Use -y to automatically confirm the one-time cache creation prompt
    gemapi request "Explain the purpose of the main.go file in this project." -y
    ```

    The output will be different. You'll see messages indicating that a cache is being created.

    ```text
    ...
    ℹ️ Caching is disabled by default
       To enable caching, add @enable-cache to your .grove/rules file
    ⚠️  ALPHA FEATURE WARNING
    ...
    🆕 No existing cache found
    💰 Creating new cache (one-time operation)...
    📤 Uploading files for cache...
      ✅ cached-context (0.12s)
    ...
      ✅ Cache created: cachedContents/1ab23c4d...
      📅 Expires in 1 hour
    ...
    📊 Token usage:
    ╭────────────────────────────────╮
    │ New Cache Tokens: 34 tokens    │
    ...
    ```
    *Note: Cache creation is a one-time operation for a given set of files. It may incur a small cost.*

3.  Now, run the exact same request a final time.

    ```bash
    gemapi request "Explain the purpose of the main.go file in this project."
    ```

    This time, `gemapi` finds the valid cache and uses it, significantly reducing the number of tokens sent in the prompt.

    ```text
    ...
    ✅ Cache is valid (expires in 59 minutes)
    ...
    📊 Token usage:
    ╭────────────────────────────────╮
    │ Cold (Cached): 34 tokens       │
    │ Hot (Dynamic): 0 tokens        │
    │ User Prompt: 14 tokens         │
    │ -------------------------------- │
    │ Total Prompt: 34 tokens        │
    ...
    │ Total API Usage: 14 tokens     │
    │ Cache Hit Rate: 100.0%         │
    ...
    ```
    Notice that the "Total API Usage" only includes the new "User Prompt" tokens. The 34 tokens from `main.go` were served from the cache, reducing API costs and latency.

### Step 4: Reviewing Your Usage

`gemapi` keeps a local log of all requests, allowing you to easily review your usage history, token counts, and costs.

1.  Run the `query local` command to see the requests you just made.

    ```bash
    gemapi query local
    ```

2.  The output is a table summarizing your recent activity.

    ```text
    Fetching local Gemini API logs for the last 24 hour(s)...

    Timestamp           Model           Repo/Branch               Caller          Cached  Prompt   Compl   Total  Cache%     Cost         Time   Status
    ------------------------------------------------------------------------------------------------------------------------------------------------
    09-26 10:32:05      gemini-2.0-flash my-go-project/main      gemapi-request      34      34     110     144  100.0%  $0.000047   0.98s  ✓
    09-26 10:31:15      gemini-2.0-flash my-go-project/main      gemapi-request       -      48     115     163       -  $0.000051   1.55s  ✓
    09-26 10:30:20      gemapi-2.0-flash my-go-project/main      gemapi-request       -      48     112     160       -  $0.000050   1.41s  ✓
    09-26 10:29:45      gemapi-2.0-flash -                         gemapi-request       -       7     123     130       -  $0.000050   1.23s  ✓
    ```

This command gives you a quick overview of your local API interactions, helping you track costs and understand token consumption patterns.

### Next Steps

You have now learned the fundamental workflow of `gemapi`. From here, you can explore more advanced features:
*   See all available commands and flags in the [Command Reference](./command-reference.md).
*   Learn more about context management in the [Core Concepts](./core-concepts.md) guide.
*   Take a deeper look at directives and manual controls in the [Caching Deep Dive](./caching-deep-dive.md).
*   Explore all observability tools in the [Observability Guide](./observability-guide.md).