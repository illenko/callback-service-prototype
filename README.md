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


```sql

INSERT INTO callback_message (id, payment_id, payload, url, created_at, updated_at, scheduled_at, delivered_at,
                              delivery_attempts, error)
SELECT gen_random_uuid()                                                AS id,
       gen_random_uuid()                                                AS payment_id,
       jsonb_build_object('id', gen_random_uuid(), 'status', 'created') AS payload,
       'http://localhost:8085/success-delayed'                          AS url,
       NOW()                                                            AS created_at,
       NOW()                                                            AS updated_at,
       NOW()                                                            AS scheduled_at,
       NULL                                                             AS delivered_at,
       0                                                                AS delivery_attempts,
       NULL                                                             AS error
FROM generate_series(1, 100000);


```