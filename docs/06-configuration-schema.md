# Grove Gemini Configuration

Schema for the 'gemini' extension in `grove.yml`.

## Properties

| Property          | Type   | Description                                                                                             |
| ----------------- | ------ | ------------------------------------------------------------------------------------------------------- |
| `api_key`         | string | A direct string value for the Gemini API key, used as a fallback if an environment variable or command is not set. |
| `api_key_command` | string | A shell command that outputs the Gemini API key to stdout, used for dynamic or secure key retrieval.      |