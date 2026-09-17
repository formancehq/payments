-- payment_id records the payment an adjustment was derived from. It stays NULL
-- when the adjustment does not reflect an actual payment, e.g. a failure the PSP
-- returned directly before any payment existed. Adjustments written before this
-- migration are all NULL: the distinction only holds from here on.
alter table payment_initiation_adjustments
    add column payment_id varchar;
