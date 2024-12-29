Payment events:

```json

{
  "id": "99d2aa54-7dc6-487e-a3eb-77a5c6135446",
  "event": "created",
  "payload": {
    "id": "e3814f7f-b6ba-4cf8-923b-f7064c8b614c",
    "amount": 100,
    "currency": "USD",
    "status": "created",
    "createdAt": "2021-09-29T12:00:00Z",
    "updatedAt": "2021-09-29T12:00:00Z",
    "callbackUrl": "http://localhost:8085/callback"
  }
}

```

```json
{
  "id": "72d4026c-f16e-4270-bbb7-6cd87ec8594d",
  "event": "updated",
  "payload": {
    "id": "e3814f7f-b6ba-4cf8-923b-f7064c8b614c",
    "amount": 100,
    "currency": "USD",
    "status": "pending",
    "createdAt": "2021-09-29T12:00:00Z",
    "updatedAt": "2021-09-29T12:00:00Z",
    "callbackUrl": "http://localhost:8085/callback"
  }
}
```

Payment statuses: created, pending, succeeded, failed

Callback body:

```json
{
  "id": "e3814f7f-b6ba-4cf8-923b-f7064c8b614c",
  "status": "succeeded"
}

```

Callback messages table: id, payment_id, payload, created_at, processed_at, error.



