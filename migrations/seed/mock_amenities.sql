-- Hall Amenities
INSERT INTO amenities (name, description)
VALUES
    ('IMAX', 'Large-format screens offering crystal-clear visuals'),
    ('Dolby Atmos Sound', 'Advanced surround sound system for immersive audio'),
    ('3D Capabilities', 'Ability to show films in 3D'),
    ('4DX Experience', 'Interactive experience with motion seats and environmental effects'),
    ('Recliner Seats', 'Comfortable reclining seats for better comfort during the movie'),
    ('VIP Lounge Access', 'Exclusive access to a VIP lounge for premium guests'),
    ('Reserved Seating', 'Seats that can be reserved in advance for select guests'),
    ('Wheelchair Accessible', 'Designated spaces for wheelchair users'),
    ('Premium Seating', 'Extra-comfortable seats with enhanced legroom or additional features'),
    ('Noise Cancelling Headsets', 'Available for patrons seeking a quieter experience'),
    ('Leg Rest', 'Automatic or manual footrests for extra comfort'),
    ('Bar Service', 'A small bar available inside the hall to order drinks'),
    ('Hearing Assistance', 'Special devices for people with hearing impairments'),
    ('Family Seating Area', 'A dedicated section in the hall for family-friendly seating');


-- Theater Amenities
INSERT INTO amenities (name, description)
VALUES
    ('Concessions Stand', 'Availability of food and drinks inside the theater'),
    ('Parking Facilities', 'Dedicated parking space for theater guests'),
    ('Restrooms', 'Clean and accessible restrooms'),
    ('Child-Friendly', 'Kid-friendly services and screenings'),
    ('Lounge Access', 'VIP lounges for relaxation before or after screenings'),
    ('Pet-Friendly', 'A theater that allows pets, possibly with special accommodations'),
    ('WiFi Access', 'Free or premium WiFi available for guests'),
    ('Mobile Ticketing', 'Ability to purchase and scan tickets using mobile devices'),
    ('Reserved Parking', 'Premium parking spots for VIP guests'),
    ('Private Screening Rooms', 'Exclusive private rooms for special screenings'),
    ('Arcade & Entertainment Zone', 'Game zones for additional fun before or after movies'),
    ('Cozy Waiting Area', 'Comfortable seating and ambiance in the waiting area'),
    ('Self-Check-In Kiosks', 'Automated kiosks for easy ticket pickup and check-in'),
    ('Merchandise Store', 'A shop selling movie-themed merchandise and collectibles'),
    ('Valet Parking', 'Valet service for a hassle-free parking experience'),
    ('Outdoor Seating Area', 'A comfortable outdoor seating space for guests'),
    ('Birthday & Event Hosting', 'Special packages for birthdays and private events');

-- Assign amenities to halls
INSERT INTO hall_amenities (hall_id, amenity_id)
VALUES
    -- IMAX Halls
    (1, (SELECT id FROM amenities WHERE name = 'IMAX')),
    (1, (SELECT id FROM amenities WHERE name = 'Premium Seating')),
    (1, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    
    (6, (SELECT id FROM amenities WHERE name = 'IMAX')),
    (6, (SELECT id FROM amenities WHERE name = 'Noise Cancelling Headsets')),
    (6, (SELECT id FROM amenities WHERE name = 'VIP Lounge Access')),

    -- Dolby Atmos Halls
    (2, (SELECT id FROM amenities WHERE name = 'Dolby Atmos Sound')),
    (2, (SELECT id FROM amenities WHERE name = 'Bar Service')),
    (2, (SELECT id FROM amenities WHERE name = 'Hearing Assistance')),

    (7, (SELECT id FROM amenities WHERE name = 'Dolby Atmos Sound')),
    (7, (SELECT id FROM amenities WHERE name = 'Bar Service')),
    (7, (SELECT id FROM amenities WHERE name = 'Leg Rest')),

    -- 3D Halls
    (3, (SELECT id FROM amenities WHERE name = '3D Capabilities')),
    (3, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (3, (SELECT id FROM amenities WHERE name = 'Family Seating Area')),

    (8, (SELECT id FROM amenities WHERE name = '3D Capabilities')),
    (8, (SELECT id FROM amenities WHERE name = 'Noise Cancelling Headsets')),

    -- 4DX Halls
    (4, (SELECT id FROM amenities WHERE name = '4DX Experience')),
    (4, (SELECT id FROM amenities WHERE name = 'Premium Seating')),
    (4, (SELECT id FROM amenities WHERE name = 'Hearing Assistance')),
    (4, (SELECT id FROM amenities WHERE name = 'Family Seating Area')),

    (9, (SELECT id FROM amenities WHERE name = '4DX Experience')),
    (9, (SELECT id FROM amenities WHERE name = 'Premium Seating')),
    (9, (SELECT id FROM amenities WHERE name = 'Leg Rest')),

    -- Standard Halls
    (5, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (5, (SELECT id FROM amenities WHERE name = 'Wheelchair Accessible')),
    (5, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),

    (10, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),
    (10, (SELECT id FROM amenities WHERE name = 'Family Seating Area')),
    (10, (SELECT id FROM amenities WHERE name = 'Bar Service')),

    (11, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (11, (SELECT id FROM amenities WHERE name = 'Wheelchair Accessible')),

    (12, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (12, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (12, (SELECT id FROM amenities WHERE name = 'Bar Service')),

    (13, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (13, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),
    (13, (SELECT id FROM amenities WHERE name = 'Leg Rest')),

    (14, (SELECT id FROM amenities WHERE name = 'Wheelchair Accessible')),
    (14, (SELECT id FROM amenities WHERE name = 'Family Seating Area')),
    (14, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),

    (15, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (15, (SELECT id FROM amenities WHERE name = 'Bar Service')),

    (16, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (16, (SELECT id FROM amenities WHERE name = 'Wheelchair Accessible')),
    (16, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),

    (17, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (17, (SELECT id FROM amenities WHERE name = 'Family Seating Area')),

    (18, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),
    (18, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (18, (SELECT id FROM amenities WHERE name = 'Bar Service')),

    (19, (SELECT id FROM amenities WHERE name = 'Wheelchair Accessible')),
    (19, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (19, (SELECT id FROM amenities WHERE name = 'Family Seating Area')),

    (20, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (20, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),

    (21, (SELECT id FROM amenities WHERE name = 'Bar Service')),
    (21, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),

    (22, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (22, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (22, (SELECT id FROM amenities WHERE name = 'Wheelchair Accessible')),

    (23, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (23, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (23, (SELECT id FROM amenities WHERE name = 'Leg Rest')),

    (24, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (24, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),
    (24, (SELECT id FROM amenities WHERE name = 'Wheelchair Accessible')),

    (25, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (25, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),

    (26, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (26, (SELECT id FROM amenities WHERE name = 'Bar Service')),
    (26, (SELECT id FROM amenities WHERE name = 'Leg Rest')),

    (27, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (27, (SELECT id FROM amenities WHERE name = 'Family Seating Area')),
    (27, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),

    (28, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (28, (SELECT id FROM amenities WHERE name = 'Bar Service')),
    (28, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),

    (29, (SELECT id FROM amenities WHERE name = 'Wheelchair Accessible')),
    (29, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),

    (30, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (30, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),

    (31, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (31, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (31, (SELECT id FROM amenities WHERE name = 'Leg Rest')),

    (32, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (32, (SELECT id FROM amenities WHERE name = 'Bar Service')),

    (33, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (33, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),

    (34, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (34, (SELECT id FROM amenities WHERE name = 'Family Seating Area')),

    (35, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (35, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (35, (SELECT id FROM amenities WHERE name = 'Bar Service')),

    (36, (SELECT id FROM amenities WHERE name = 'Wheelchair Accessible')),
    (36, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),

    (37, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (37, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),

    (38, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (38, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),

    (39, (SELECT id FROM amenities WHERE name = 'Bar Service')),
    (39, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),

    (40, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (40, (SELECT id FROM amenities WHERE name = 'Wheelchair Accessible')),

    (41, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (41, (SELECT id FROM amenities WHERE name = 'Family Seating Area')),

    (42, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (42, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),

    (43, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (43, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),

    (44, (SELECT id FROM amenities WHERE name = 'Bar Service')),
    (44, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),

    (45, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),
    (45, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),

    (46, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (46, (SELECT id FROM amenities WHERE name = 'Family Seating Area')),

    (47, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (47, (SELECT id FROM amenities WHERE name = 'Wheelchair Accessible')),

    (48, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),
    (48, (SELECT id FROM amenities WHERE name = 'Bar Service')),

    (49, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (49, (SELECT id FROM amenities WHERE name = 'Reserved Seating')),

    (50, (SELECT id FROM amenities WHERE name = 'Recliner Seats')),
    (50, (SELECT id FROM amenities WHERE name = 'Family Seating Area'));


-- Assign amenities to theaters
INSERT INTO theater_amenities (theater_id, amenity_id)
VALUES
    -- Theater 1
    (1, (SELECT id FROM amenities WHERE name = 'Parking Facilities')),
    (1, (SELECT id FROM amenities WHERE name = 'Restrooms')),
    (1, (SELECT id FROM amenities WHERE name = 'Lounge Access')),
    (1, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),

    -- Theater 2
    (2, (SELECT id FROM amenities WHERE name = 'Child-Friendly')),
    (2, (SELECT id FROM amenities WHERE name = 'Pet-Friendly')),
    (2, (SELECT id FROM amenities WHERE name = 'Parking Facilities')),
    
    -- Theater 3
    (3, (SELECT id FROM amenities WHERE name = 'Parking Facilities')),
    (3, (SELECT id FROM amenities WHERE name = 'Restrooms')),
    (3, (SELECT id FROM amenities WHERE name = 'Lounge Access')),
    (3, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (3, (SELECT id FROM amenities WHERE name = 'WiFi Access')),

    -- Theater 4
    (4, (SELECT id FROM amenities WHERE name = 'Restrooms')),
    (4, (SELECT id FROM amenities WHERE name = 'Child-Friendly')),
    (4, (SELECT id FROM amenities WHERE name = 'Lounge Access')),

    -- Theater 5
    (5, (SELECT id FROM amenities WHERE name = 'Parking Facilities')),
    (5, (SELECT id FROM amenities WHERE name = 'Pet-Friendly')),
    (5, (SELECT id FROM amenities WHERE name = 'Restrooms')),
    (5, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (5, (SELECT id FROM amenities WHERE name = 'WiFi Access')),

    -- Theater 6
    (6, (SELECT id FROM amenities WHERE name = 'IMAX')),
    (6, (SELECT id FROM amenities WHERE name = '3D Capabilities')),
    (6, (SELECT id FROM amenities WHERE name = 'Dolby Atmos Sound')),

    -- Theater 7
    (7, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (7, (SELECT id FROM amenities WHERE name = 'Parking Facilities')),
    (7, (SELECT id FROM amenities WHERE name = 'Lounge Access')),

    -- Theater 8
    (8, (SELECT id FROM amenities WHERE name = '4DX Experience')),
    (8, (SELECT id FROM amenities WHERE name = '3D Capabilities')),
    (8, (SELECT id FROM amenities WHERE name = 'Dolby Atmos Sound')),
    (8, (SELECT id FROM amenities WHERE name = 'IMAX')),
    
    -- Theater 9
    (9, (SELECT id FROM amenities WHERE name = 'WiFi Access')),
    (9, (SELECT id FROM amenities WHERE name = 'Lounge Access')),
    (9, (SELECT id FROM amenities WHERE name = 'Child-Friendly')),

    -- Theater 10
    (10, (SELECT id FROM amenities WHERE name = 'Pet-Friendly')),
    (10, (SELECT id FROM amenities WHERE name = 'Restrooms')),
    (10, (SELECT id FROM amenities WHERE name = 'Concessions Stand')),
    (10, (SELECT id FROM amenities WHERE name = 'Parking Facilities'));
