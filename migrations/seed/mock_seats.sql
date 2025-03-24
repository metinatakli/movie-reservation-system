-- Premium seat layouts
-- Insert seats for halls with 'Recliner Seats' amenity (6x8 layout)
WITH recliner_halls AS (
    SELECT h.id
    FROM hall_amenities ha
    JOIN halls h ON h.id = ha.hall_id
    JOIN amenities a ON a.id = ha.amenity_id
    WHERE a.name = 'Recliner Seats'
),
seat_rows AS (
    SELECT generate_series(1, 6) AS row
),
seat_cols AS (
    SELECT generate_series(1, 8) AS col
)
INSERT INTO seats (hall_id, seat_row, seat_col, seat_type, extra_price)
SELECT 
    rh.id AS hall_id, 
    r.row AS seat_row, 
    c.col AS seat_col, 
    'Recliner' AS seat_type,
    10.99 AS extra_price
FROM recliner_halls rh
CROSS JOIN seat_rows r
CROSS JOIN seat_cols c;

-- Insert IMAX/4DX Seats
WITH imax_halls AS (
    SELECT h.id
    FROM hall_amenities ha
    JOIN halls h ON h.id = ha.hall_id
    JOIN amenities a ON a.id = ha.amenity_id
    WHERE a.name IN ('IMAX', '4DX Experience')
),
seat_rows AS (
    SELECT generate_series(1, 8) AS row
),
seat_cols AS (
    SELECT generate_series(1, 12) AS col
)
INSERT INTO seats (hall_id, seat_row, seat_col, seat_type)
SELECT 
    ih.id AS hall_id, 
    r.row AS seat_row, 
    c.col AS seat_col, 
    'Standard' AS seat_type
FROM imax_halls ih
CROSS JOIN seat_rows r
CROSS JOIN seat_cols c;

-- Insert Dolby Atmos Seats
WITH dolby_halls AS (
    SELECT h.id
    FROM hall_amenities ha
    JOIN halls h ON h.id = ha.hall_id
    JOIN amenities a ON a.id = ha.amenity_id
    WHERE a.name = 'Dolby Atmos Sound'
),
seat_rows AS (
    SELECT generate_series(1, 10) AS row
),
seat_cols AS (
    SELECT generate_series(1, 14) AS col
)
INSERT INTO seats (hall_id, seat_row, seat_col, seat_type)
SELECT 
    dh.id AS hall_id, 
    r.row AS seat_row, 
    c.col AS seat_col, 
    'Standard' AS seat_type
FROM dolby_halls dh
CROSS JOIN seat_rows r
CROSS JOIN seat_cols c;

-- Small Hall Seats (5x10 layout)
WITH small_halls AS (
    SELECT id 
    FROM halls
    WHERE id IN (3,5,8,13,16,17,20,22,25,26,30,33,34,36,37,38,40,41,43,45)
),
seat_rows AS (
    SELECT generate_series(1, 5) AS row
),
seat_cols AS (
    SELECT generate_series(1, 10) AS col
)
INSERT INTO seats (hall_id, seat_row, seat_col, seat_type)
SELECT 
    sh.id AS hall_id, 
    r.row AS seat_row, 
    c.col AS seat_col, 
    'Standard' AS seat_type
FROM small_halls sh
CROSS JOIN seat_rows r
CROSS JOIN seat_cols c;

-- Medium Hall Seats (10x15 layout)
WITH medium_halls AS (
    SELECT id 
    FROM halls
    WHERE id IN (12,15,18,21,24,27,29,32,39,42,44,47,48,49,50)
),
seat_rows AS (
    SELECT generate_series(1, 10) AS row
),
seat_cols AS (
    SELECT generate_series(1, 15) AS col
)
INSERT INTO seats (hall_id, seat_row, seat_col, seat_type)
SELECT 
    mh.id AS hall_id, 
    r.row AS seat_row, 
    c.col AS seat_col, 
    'Standard' AS seat_type
FROM medium_halls mh
CROSS JOIN seat_rows r
CROSS JOIN seat_cols c;

-- Large Hall Seats (15x20 layout)
WITH large_halls AS (
    SELECT id 
    FROM halls
    WHERE id IN (11,14,19,23,28,31,35,46)
),
seat_rows AS (
    SELECT generate_series(1, 15) AS row
),
seat_cols AS (
    SELECT generate_series(1, 20) AS col
)
INSERT INTO seats (hall_id, seat_row, seat_col, seat_type)
SELECT 
    lh.id AS hall_id, 
    r.row AS seat_row, 
    c.col AS seat_col, 
    'Standard' AS seat_type
FROM large_halls lh
CROSS JOIN seat_rows r
CROSS JOIN seat_cols c;

-- VIP Seats
UPDATE seats
SET seat_type = 'VIP', extra_price = 15.99 
WHERE seat_row = 1
AND hall_id IN (
    SELECT hall_id 
    FROM seats 
    GROUP BY hall_id 
    ORDER BY random() 
    LIMIT 15
);