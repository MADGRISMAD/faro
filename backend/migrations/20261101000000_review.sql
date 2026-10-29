-- Lo que la revisión automática encontró en el artículo (cifras sin fuente, promesas absolutas…): se muestra al editarlo
ALTER TABLE articles ADD COLUMN IF NOT EXISTS review_notes jsonb NOT NULL DEFAULT '[]';
