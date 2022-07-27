CREATE TABLE `vs` (
  `id` CHAR(64) PRIMARY KEY
);
CREATE TABLE `es` (
  `from` CHAR(64) NOT NULL REFERENCES `vs` (id),
  `to` CHAR(64) NOT NULL REFERENCES `vs` (id),
  `eweight` INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (`from`, `to`)
);
-- Warning: In `alter.sql` there are schema changes. Refer it for final schema.
