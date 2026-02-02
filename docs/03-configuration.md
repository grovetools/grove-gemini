## Gemini Settings

| Property | Description |
| :--- | :--- |
| `api_key` | (string, optional) The raw API key string used to authenticate with Google's Gemini API. While supported, storing API keys in plain text configuration files is discouraged. This value is used only if the `GEMINI_API_KEY` environment variable is not set and `api_key_command` is not configured. |
| `api_key_command` | (string, optional) A shell command to execute to retrieve the API key. This property allows for secure integration with password managers (e.g., `1op`, `pass`, `lpass`) or other secret stores. The command should output the key to stdout. This takes precedence over the static `api_key` field. |

```yaml
gemini:
  # Securely fetch the API key using a password manager CLI
  api_key_command: "op read op://Private/Gemini/credential"
  
  # Or set directly (not recommended for version-controlled files)
  # api_key: "AIzaSy..."
```