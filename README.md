Time Clock
===

# API

## Attach

Attach a user to scheduler.

### Endpoint

`POST` <https://time-clock.epoch.tw/api/attach>

### Actions

| Name          | Description                   |
| ------------- | ----------------------------- |
| `ISSUE_TOKEN` | Issue a new token to calendar |
| `CLOCK_IN`    | Clock in                      |
| `CLOCK_OUT`   | Clock out                     |

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
      "action": "ISSUE_TOKEN",
      "date": "2021-06-01T18:00:00+08:00"
    }
  ],
  "id": "<USER_ID>"
}
```

### Response

`201 Created` | `200 OK`

## Detach

Detach a user from scheduler.

### Endpoint

`POST` <https://time-clock.epoch.tw/api/detach>

### Request

```json
{
  "credentials": {
    "username": "<USERNAME>"
  },
  "token": "<TOKEN>"
}
```

### Response

`204 No Content`

## Verify

Verify a user.

### Endpoint

`POST` <https://time-clock.epoch.tw/api/verify>

### Request

```json
{
  "credentials": {
    "username": "<USERNAME>"
  },
  "token": "<TOKEN>"
}
```

### Response

`200 OK`
