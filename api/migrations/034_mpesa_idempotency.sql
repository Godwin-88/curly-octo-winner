-- Migration: 034_mpesa_idempotency.sql
-- Description: Guarantee at-most-one payment per Daraja CheckoutRequestID so
-- retried M-Pesa STK callbacks can never create or apply a payment twice.
-- The application layer already treats non-pending payments as processed;
-- this index makes duplicate pending rows impossible in the first place.

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_mpesa_checkout
    ON payments (checkout_request_id)
    WHERE channel = 'mpesa' AND checkout_request_id IS NOT NULL;
