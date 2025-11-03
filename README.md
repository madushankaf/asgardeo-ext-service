# Pre-Issue Access Token Extension Service

Simple PoC extension service for WSO2 Asgardeo pre-issue access token hook.

## Run

```bash
go run main.go
```

Or build and run:
```bash
go build -o service main.go
./service
```

The service listens on port 8080 by default (set `PORT` env var to change).

## Endpoint

POST `/token`

## Example Request

Minimal request format:
```json
{
    "actionType": "PRE_ISSUE_ACCESS_TOKEN",
    "event": {
        "request": {
            "clientId": "1u31N7of6gCNR9FqkG1neSlsF_Qa",
            "grantType": "authorization_code"
        },
        "accessToken": {
            "scopes": ["openid", "profile"],
            "claims": [
                {"name": "sub", "value": "user123"}
            ]
        }
    }
}
```

## Response

Returns the same event structure (modify in code as needed for your PoC).
