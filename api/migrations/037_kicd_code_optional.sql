-- Make the KICD code genuinely optional.
--
-- The curriculum forms offer the code as optional and send "" when it is
-- blank, but kicd_code was NOT NULL and carried a UNIQUE constraint. The first
-- blank row therefore occupied "" and every later blank row in the same group
-- was rejected as a duplicate -- in practice that allowed exactly one
-- sub-strand per strand, one strand per area and one learning area per school
-- to be created without a code.
--
-- Storing "not known yet" as NULL fixes it: NULLs are distinct within a
-- unique index, so any number of code-less rows can coexist while real codes
-- stay unique. The Go layer maps "" to NULL on write and NULL back to "" on
-- read, so the JSON contract is unchanged.

ALTER TABLE learning_areas     ALTER COLUMN kicd_code DROP NOT NULL;
ALTER TABLE strands            ALTER COLUMN kicd_code DROP NOT NULL;
ALTER TABLE sub_strands        ALTER COLUMN kicd_code DROP NOT NULL;
ALTER TABLE core_competencies  ALTER COLUMN kicd_code DROP NOT NULL;
ALTER TABLE values             ALTER COLUMN kicd_code DROP NOT NULL;

UPDATE learning_areas     SET kicd_code = NULL WHERE kicd_code = '';
UPDATE strands            SET kicd_code = NULL WHERE kicd_code = '';
UPDATE sub_strands        SET kicd_code = NULL WHERE kicd_code = '';
UPDATE core_competencies  SET kicd_code = NULL WHERE kicd_code = '';
UPDATE values             SET kicd_code = NULL WHERE kicd_code = '';
