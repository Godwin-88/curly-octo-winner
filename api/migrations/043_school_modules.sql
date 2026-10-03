-- 043_school_modules.sql
-- Which modules a school has. Modules are sold separately, so the API refuses
-- a module a school does not have and the menus do not offer it.
--
-- NULL means every module: the schools that existed before modules were sold
-- separately keep everything they had. An array, even an empty one, is exactly
-- the list.

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS modules TEXT[];
