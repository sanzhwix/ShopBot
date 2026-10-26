-- Product photos: a path like /products/nan-and-co/croissant.jpg (seeded) or /uploads/p12-ab.jpg (owner upload).
-- photo_credit holds the author and license of an openly licensed photo.
ALTER TABLE products ADD COLUMN photo TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN photo_credit TEXT NOT NULL DEFAULT '';
