UPDATE payments
SET stripe_checkout_session_id = ''
WHERE stripe_checkout_session_id IS NULL;

ALTER TABLE payments
ALTER COLUMN stripe_checkout_session_id SET NOT NULL;