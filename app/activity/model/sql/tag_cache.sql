-- tag_cache 标签缓存表
-- 用途：从用户服务同步的标签数据本地副本
-- 同步策略：MQ 实时 + 定时全量兜底
-- 数据归属：用户服务是 Master，本表是只读副本

CREATE TABLE `tag_cache` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '标签ID（与用户服务 interest_tag.tag_id 一致）',
    `name` VARCHAR(50) NOT NULL COMMENT '标签名称',
    `color` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '标签颜色（如 #FF6B6B）',
    `icon` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '标签图标URL',
    `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态: 1启用 0禁用',
    `description` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '标签描述',
    `synced_at` BIGINT NOT NULL COMMENT '最后同步时间戳（秒）',
    `created_at` BIGINT NOT NULL COMMENT '原始创建时间戳（秒）',
    `updated_at` BIGINT NOT NULL COMMENT '原始更新时间戳（秒）',
    PRIMARY KEY (`id`),
    INDEX `idx_status` (`status`),
    INDEX `idx_synced_at` (`synced_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='标签缓存表（从用户服务同步）';

-- 初始化数据（可选，用于测试）
-- INSERT INTO `tag_cache` (`id`, `name`, `color`, `icon`, `status`, `description`, `synced_at`, `created_at`, `updated_at`) VALUES
-- (1, '运动', '#FF6B6B', '', 1, '运动相关活动', UNIX_TIMESTAMP(), UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
-- (2, '音乐', '#4ECDC4', '', 1, '音乐相关活动', UNIX_TIMESTAMP(), UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
-- (3, '学习', '#45B7D1', '', 1, '学习相关活动', UNIX_TIMESTAMP(), UNIX_TIMESTAMP(), UNIX_TIMESTAMP());
