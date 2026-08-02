# go-llm-proxy

## Getting Started

### Clone repo

```bash
git clone --bare https://github.com/seanrobmerriam/go-llm-proxy.git .bare
```
### Export env variables

```bash
export LLM_UPSTREAM_URL="https://api.openai.com"
export LLM_UPSTREAM_API_KEY="actual-provider-key"
export PROXY_API_KEY="client-facing-proxy-key"
```
### Run the proxy

```bash
go run .
```

### Send a Request

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer client-facing-proxy-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "your-model",
    "messages": [
      {
        "role": "user",
        "content": "Hello"
      }
    ]
  }'

```
The proxy will forward all requests to the actual model endpoint using the real api key.

## Development

### Clone this repo

```bash
git clone --bare https://github.com/seanrobmerriam/go-llm-proxy.git .bare

echo "gitdir: ./.bare" > .git

git config remote.origin.fetch "+refs/heads/*:refs/remotes/origin/*"

git worktree add main
````

### Create new worktree for your changes

```bash
git worktree add feature-name

cd feature-name

```
