-- 0751: Add social links column to actors
ALTER TABLE actors ADD COLUMN links TEXT DEFAULT '[]';
