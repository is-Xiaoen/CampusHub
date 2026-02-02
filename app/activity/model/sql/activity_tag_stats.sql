-- activity_tag_stats 活动标签统计表
-- 用途：统计活动服务维度的标签使用情况
-- 更新时机：活动创建/删除/标签变更时
-- 数据归属：活动服务独立维护，与用户服务的 usage_count 分开

CREATE TABLE `activity_tag_stats` (
    `tag_id` BIGINT UNSIGNED NOT NULL COMMENT '标签ID',
    `activity_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关联的活动数量',
    `view_count` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '标签关联活动的总浏览量（预留）',
    `updated_at` BIGINT NOT NULL COMMENT '更新时间戳（秒）',
    PRIMARY KEY (`tag_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='活动标签统计表';
