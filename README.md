FemasHR Puncher
===

# API

## Attach

Attch a user to scheduler.

### Endpoint

`POST` <https://femashr-puncher.epoch.tw/api/attach>

### Action

| Action      | Description                 |
| ----------- | --------------------------- |
| `TEST`      | create an event to calendar |
| `PUNCH_IN`  | punch in                    |
| `PUNCH_OUT` | punch out                   |

### Request

```json
{
  "company": "<COMPANY_CODE>",
  "credentials": {
    "username": "<USERNAME>",
    "password": "<PASSWORD>"
  },
  "email": "<EMAIL>",
  "events": [
    {
      "action": "TEST",
      "date": "2021-06-01T18:00:00+08:00"
    }
  ],
  "id": "<USER_ID>"
}
```

## Detach

Detach a user from scheduler.

### Endpoint

`POST` <https://femashr-puncher.epoch.tw/api/detach>

### Request

```json
{
  "credentials": {
    "username": "<USERNAME>",
    "password": "<PASSWORD>"
  }
}
```

### Response

```json
{}
```
