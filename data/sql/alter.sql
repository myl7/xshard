CREATE TABLE `vs_add_seq` (
  `seq` INTEGER PRIMARY KEY AUTOINCREMENT,
  `id` CHAR(64) UNIQUE NOT NULL
);
INSERT INTO `vs_add_seq` (`id`) SELECT `id` FROM `vs`;
DROP TABLE `vs`;
ALTER TABLE `vs_add_seq` RENAME TO `vs`;

CREATE INDEX `es_from` ON `es` (`from`);
CREATE INDEX `es_to` ON `es` (`to`);

ALTER TABLE `vs` ADD `vweight` INTEGER NOT NULL DEFAULT 0;
ALTER TABLE `vs` ADD `neighbor_num` INTEGER NOT NULL DEFAULT 0;
UPDATE `vs` SET
  `vweight` = (SELECT COALESCE(SUM(`eweight`), 0) FROM `es` WHERE `es`.`from` = `vs`.`id` AND `es`.`to` != `vs`.`id`)
    + (SELECT COALESCE(SUM(`eweight`), 0) FROM `es` WHERE `es`.`to` = `vs`.`id` AND `es`.`from` != `vs`.`id`)
    + (SELECT COALESCE(SUM(`eweight`), 0) FROM `es` WHERE `es`.`to` = `vs`.`id` AND `es`.`from` = `vs`.`id`),
  `neighbor_num` = (SELECT COUNT(*) FROM (
    SELECT `to` FROM `es` WHERE `es`.`from` = `vs`.`id` AND `es`.`to` != `vs`.`id`
      UNION SELECT `from` FROM `es` WHERE `es`.`to` = `vs`.`id` AND `es`.`from` != `vs`.`id`
  ));
