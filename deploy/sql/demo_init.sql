-- ============================================================================
-- Demo 服务数据库初始化脚本
-- ============================================================================
-- 说明：这是 Demo 示例服务使用的表，仅用于演示和学习
-- 执行：mysql -u manpao -p campushub_main < demo_init.sql
-- ============================================================================

USE campushub_main;

-- 示例表：items
CREATE TABLE IF NOT EXISTS `items` (
    `id` BIGINT NOT NULL COMMENT '主键ID（雪花算法）',
    `name` VARCHAR(100) NOT NULL COMMENT '名称',
    `description` VARCHAR(500) DEFAULT '' COMMENT '描述',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1-正常 2-禁用',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME DEFAULT NULL COMMENT '删除时间（软删除）',
    PRIMARY KEY (`id`),
    INDEX `idx_status` (`status`),
    INDEX `idx_created_at` (`created_at`),
    INDEX `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='示例资源表';

-- 插入测试数据
INSERT INTO `items` (`id`, `name`, `description`, `status`) VALUES
(1, '测试项目1', '这是第一个测试项目', 1),
(2, '测试项目2', '这是第二个测试项目', 1),
(3, '测试项目3', '这是第三个测试项目（已禁用）', 2);
