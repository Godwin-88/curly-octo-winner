-- dev_finance.sql — LOCAL DEVELOPMENT ONLY. Never run against production.
-- Gives the demo school the paybill the local Safaricom stand-in "pays" into,
-- so a paybill payment posted to devstub finds its school.
UPDATE tenants SET mpesa_shortcode = '174379'
WHERE id = 'a0000000-0000-0000-0000-000000000001';
