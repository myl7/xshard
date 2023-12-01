UPDATE `vs` SET (`w`, `n`) = (SELECT SUM(`w`), COUNT(*) FROM `es` WHERE `from` = `vs`.`id` OR `to` = `vs`.`id`);
UPDATE `vs` SET `seq` = `vs_seq`.`seq` FROM (
  SELECT `id`, ROW_NUMBER() OVER (ORDER BY `id`) - 1 AS `seq` FROM `vs`
) AS `vs_seq` WHERE `vs_seq`.`id` = `vs`.`id`;
