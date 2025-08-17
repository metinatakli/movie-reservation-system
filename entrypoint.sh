#!/bin/sh
./api \
  -port="$PORT" \
  -env="$ENV" \
  -db-dsn="$DB_DSN" \
  -redis-url="$REDIS_URL" \
  -smtp-username="$SMTP_USERNAME" \
  -smtp-password="$SMTP_PASSWORD" \
  -stripe-key="$STRIPE_KEY" \
  -stripe-webhook-secret="$STRIPE_WEBHOOK_SECRET" \
  -otel-collector-url="$OTEL_COLLECTOR_URL"
