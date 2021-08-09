Time Clock
===

# API

## Attach

Attach a user to scheduler.

### Endpoint

`POST /api/attach`

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
  "email": "<EMAIL>",
  "events": [
    {
      "action": "ISSUE_TOKEN",
      "date": "2021-06-01T18:00:00+08:00"
    }
  ],
  "id": "<USER_ID>",
  "password": "<PASSWORD>",
  "username": "<USERNAME>"
}
```

### Response

`201 Created` | `200 OK`

## Detach

Detach a user from scheduler.

### Endpoint

`POST /api/detach`

### Request

```json
{
  "token": "<TOKEN>",
  "username": "<USERNAME>"
}
```

### Response

`204 No Content`

## Verify

Verify a user.

### Endpoint

`POST /api/verify`

### Request

```json
{
  "token": "<TOKEN>",
  "username": "<USERNAME>"
}
```

### Response

`200 OK`
