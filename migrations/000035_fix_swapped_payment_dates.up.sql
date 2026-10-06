-- One-off data fix: 49 tuition payments from the bulk load on 2026-09-25
-- 22:23 IST (Freedom World School Bhatkhedi, Rs 1,53,700) had day and month
-- swapped -- e.g. 12 Sep 2026 saved as 9 Dec 2026 -- which put them in the
-- future. Each row below is (payment id, saved date, corrected date). A row is
-- changed only if it still has the saved date, and every change is recorded
-- so the down migration reverts exactly those rows.

CREATE TABLE fee_payment_date_fix_000035 (
    payment_id UUID PRIMARY KEY REFERENCES fee_payments(id) ON DELETE CASCADE,
    old_date   DATE NOT NULL,
    new_date   DATE NOT NULL
);

WITH fixes(payment_id, old_date, new_date) AS (VALUES
    ('01cde171-da25-4326-8145-4c8d911346db'::uuid, DATE '2026-10-07', DATE '2026-07-10'),
    ('1a628620-dde5-4244-b322-912a3186046e'::uuid, DATE '2026-10-07', DATE '2026-07-10'),
    ('dfb43f8a-846d-459f-b647-dfd80c11e995'::uuid, DATE '2026-10-07', DATE '2026-07-10'),
    ('f4673cb0-132c-4e50-af50-bd74ee65de55'::uuid, DATE '2026-10-07', DATE '2026-07-10'),
    ('2901ef87-f448-4ef6-93d6-3572c44ec27b'::uuid, DATE '2026-10-08', DATE '2026-08-10'),
    ('68c26c8e-f90a-4160-8ee5-fb376af2dfc5'::uuid, DATE '2026-10-08', DATE '2026-08-10'),
    ('877e1083-20e9-4eb9-a25f-3aae93090b40'::uuid, DATE '2026-10-08', DATE '2026-08-10'),
    ('a158a537-4eca-460c-a61e-1ae43e3904fc'::uuid, DATE '2026-10-08', DATE '2026-08-10'),
    ('a59c56c7-d10e-499b-91d1-8128b00dc01f'::uuid, DATE '2026-10-08', DATE '2026-08-10'),
    ('fe96f330-dbab-446c-b2e5-a0e292be39fc'::uuid, DATE '2026-10-08', DATE '2026-08-10'),
    ('135d5702-55a4-4078-9b27-6ee27a5fe1a3'::uuid, DATE '2026-10-09', DATE '2026-09-10'),
    ('2f12e410-89c5-42c9-b29c-674e36cb3d77'::uuid, DATE '2026-10-09', DATE '2026-09-10'),
    ('4a9a0f4f-673b-4ddd-8f42-071d44723b44'::uuid, DATE '2026-10-09', DATE '2026-09-10'),
    ('87547901-b1e7-458f-96f6-5f6667f0a686'::uuid, DATE '2026-10-09', DATE '2026-09-10'),
    ('ddb32b74-51d9-4753-9778-a0dc5b682047'::uuid, DATE '2026-10-09', DATE '2026-09-10'),
    ('e6f15f55-91fe-4a88-b1e3-78df7eb6f073'::uuid, DATE '2026-10-09', DATE '2026-09-10'),
    ('217dc2a8-768c-4be3-9ec5-cb4522c52562'::uuid, DATE '2026-11-07', DATE '2026-07-11'),
    ('50f430ff-ff45-4933-99e7-6188259e2572'::uuid, DATE '2026-11-07', DATE '2026-07-11'),
    ('7b96884d-a88d-46d6-939d-9ea2c7bcff6f'::uuid, DATE '2026-11-07', DATE '2026-07-11'),
    ('010f1d25-95dd-4bd8-92c1-b8f1193867d4'::uuid, DATE '2026-11-08', DATE '2026-08-11'),
    ('c0219a55-e8e3-4b6e-a140-df2e2965b862'::uuid, DATE '2026-11-08', DATE '2026-08-11'),
    ('e6de7dcd-398a-4ed7-947b-eb1d123fab5b'::uuid, DATE '2026-11-08', DATE '2026-08-11'),
    ('1835a6d3-2385-47d5-bd24-ba8da9382710'::uuid, DATE '2026-11-09', DATE '2026-09-11'),
    ('3cf857df-6c5c-40dd-a171-b40a327cb1a2'::uuid, DATE '2026-11-09', DATE '2026-09-11'),
    ('cf55d5c8-78d8-4e10-bf0b-37ab9387e958'::uuid, DATE '2026-11-09', DATE '2026-09-11'),
    ('d82d7326-618f-4f06-877b-d8a0ae07f108'::uuid, DATE '2026-11-09', DATE '2026-09-11'),
    ('e6bca485-5d6d-4ec7-8f97-4efe5991f9f7'::uuid, DATE '2026-11-09', DATE '2026-09-11'),
    ('298d09cd-f47c-4a66-b7ba-6a5f9dc8c30f'::uuid, DATE '2026-12-06', DATE '2026-06-12'),
    ('73c2a6b5-aec3-40fe-a2bd-a93497d30090'::uuid, DATE '2026-12-06', DATE '2026-06-12'),
    ('2194b8d5-efd5-4d57-8e0c-111df6b27cc9'::uuid, DATE '2026-12-08', DATE '2026-08-12'),
    ('48299713-a513-4d30-beaf-110c7af2b185'::uuid, DATE '2026-12-08', DATE '2026-08-12'),
    ('4e88a73c-86ba-4d6a-a113-8dcd7f3d3c33'::uuid, DATE '2026-12-08', DATE '2026-08-12'),
    ('555c9b4d-a8fd-4e1b-805a-25d1c9416edd'::uuid, DATE '2026-12-08', DATE '2026-08-12'),
    ('69534dff-03de-4513-a08e-2f7446eca704'::uuid, DATE '2026-12-08', DATE '2026-08-12'),
    ('ea49eea8-ec75-409a-aa99-c1066ec0abfd'::uuid, DATE '2026-12-08', DATE '2026-08-12'),
    ('13774a30-20f8-40c3-9040-e1d3ad83f241'::uuid, DATE '2026-12-09', DATE '2026-09-12'),
    ('171cfcfa-5140-4f83-9374-7c69f761d006'::uuid, DATE '2026-12-09', DATE '2026-09-12'),
    ('196cc8ae-07cf-4a26-8cf9-887a923bc616'::uuid, DATE '2026-12-09', DATE '2026-09-12'),
    ('3458c1f9-3025-4283-8b92-27ece42bed83'::uuid, DATE '2026-12-09', DATE '2026-09-12'),
    ('36c49ec1-a62a-4324-a59d-35bd0c5a9c54'::uuid, DATE '2026-12-09', DATE '2026-09-12'),
    ('5e49989e-b02e-45a8-92c2-e925c7e11120'::uuid, DATE '2026-12-09', DATE '2026-09-12'),
    ('6e5fca4f-40f2-463b-bff5-5e4417fcb601'::uuid, DATE '2026-12-09', DATE '2026-09-12'),
    ('70cbc492-d8e4-4eeb-a307-903901d40b8a'::uuid, DATE '2026-12-09', DATE '2026-09-12'),
    ('717e39db-751b-426e-89ac-1dc3d160262a'::uuid, DATE '2026-12-09', DATE '2026-09-12'),
    ('7d96451e-64e7-4d52-ad2c-83c8e4a20263'::uuid, DATE '2026-12-09', DATE '2026-09-12'),
    ('9065fc2b-0c59-4195-854c-d1c6b1db9068'::uuid, DATE '2026-12-09', DATE '2026-09-12'),
    ('bc9b058c-3e70-4215-8000-d6bd0b0e32e8'::uuid, DATE '2026-12-09', DATE '2026-09-12'),
    ('cc2722c1-68ba-4aae-a49e-4de04e001975'::uuid, DATE '2026-12-09', DATE '2026-09-12'),
    ('f7939405-67d2-4df8-bfa0-71d41ae608e7'::uuid, DATE '2026-12-09', DATE '2026-09-12')
),
updated AS (
    UPDATE fee_payments fp
    SET payment_date = f.new_date
    FROM fixes f
    WHERE fp.id = f.payment_id AND fp.payment_date = f.old_date AND NOT fp.voided
    RETURNING fp.id, f.old_date, f.new_date
)
INSERT INTO fee_payment_date_fix_000035 (payment_id, old_date, new_date)
SELECT id, old_date, new_date FROM updated;
