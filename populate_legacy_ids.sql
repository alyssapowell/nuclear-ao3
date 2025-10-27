-- Populate existing works with legacy IDs to simulate AO3 migration
-- These represent works that would have existing AO3 numeric IDs

-- Update works with realistic legacy AO3 IDs (typical range: 100000 - 50000000)

-- First batch: Higher legacy IDs (newer works on original AO3)
UPDATE works SET legacy_id = 42876543 WHERE id = '79511f31-787a-483c-bcf5-0447ec907d54'; -- Midnight Hearts
UPDATE works SET legacy_id = 41923847 WHERE id = 'c07055e5-01b2-4d5b-9fcf-2aa7ada020af'; -- Whispered Hearts  
UPDATE works SET legacy_id = 40876234 WHERE id = '78c58e3b-9662-42e5-a5db-7c95c449b892'; -- Forgotten Dances
UPDATE works SET legacy_id = 39847562 WHERE id = 'a86a5247-b24b-4170-b84e-e25ffdcabb21'; -- Dancing with Wishs
UPDATE works SET legacy_id = 38923456 WHERE id = '1391fd80-2623-4e48-b430-5ab218c8a317'; -- How Severus protects

-- Second batch: Mid-range legacy IDs  
UPDATE works SET legacy_id = 25847392 WHERE id = 'cc77f74c-20e2-49c2-84c3-195b6ab3018e'; -- Sacred Promises
UPDATE works SET legacy_id = 23947583 WHERE id = '745b0467-a906-4e84-9438-bcd9647403f4'; -- The Ancient Promise
UPDATE works SET legacy_id = 22847593 WHERE id = '4f6af916-166f-4f31-ba48-9c0d76a5617b'; -- Hope and Hope
UPDATE works SET legacy_id = 21847294 WHERE id = 'b9e9b5da-e62a-484f-bccb-8e0fa6c4748c'; -- Strong Loves
UPDATE works SET legacy_id = 20847583 WHERE id = '8bbf13b5-f93d-4e8c-89ce-f70a0bdf1639'; -- A Tender Mind

-- Third batch: Lower legacy IDs (older works on original AO3)
UPDATE works SET legacy_id = 9847392 WHERE id = 'cb188aeb-cd96-4c2f-a5df-c9088e8c444e'; -- The Mystery of Luna
UPDATE works SET legacy_id = 8847583 WHERE id = '86f9284b-be9e-4c01-9410-2a1243f8054f'; -- The Fear Chronicles
UPDATE works SET legacy_id = 7847294 WHERE id = 'a21123c7-c8d2-4d2c-ae12-84b88ae0d56b'; -- Broken Hearts
UPDATE works SET legacy_id = 6847583 WHERE id = '8aad307a-35c9-447a-928f-163761292d00'; -- Beyond the Road
UPDATE works SET legacy_id = 5847392 WHERE id = 'dd7ba75f-e2e5-42e3-85db-b6067a2caa54'; -- How Harry whispers

-- Fourth batch: Very old legacy IDs (earliest AO3 works)
UPDATE works SET legacy_id = 847392 WHERE id = 'd51a53a8-0e8b-453a-8df2-cb9e7b2e0276'; -- Fear in Another World
UPDATE works SET legacy_id = 547583 WHERE id = '08a425a9-9ba2-4a6a-903f-fc89d25dd04d'; -- Forbidden Hearts
UPDATE works SET legacy_id = 347294 WHERE id = 'bbb3210e-98eb-4a26-a4de-094ad8401591'; -- Barty's Tale
UPDATE works SET legacy_id = 247583 WHERE id = '24d6cf83-b6fb-4b6e-a9b1-e2b1699d5fb7'; -- Sirius and the Brave Kiss
UPDATE works SET legacy_id = 147392 WHERE id = 'de163050-a434-4ae2-ab5b-835e56a4f864'; -- Wish in the Library

-- Leave some works without legacy IDs to simulate new Nuclear AO3 works
-- (The first 2 works we already populated: 12345, 67890)

-- Verify the population
SELECT 
    legacy_id,
    title,
    id
FROM works 
WHERE legacy_id IS NOT NULL 
ORDER BY legacy_id DESC;