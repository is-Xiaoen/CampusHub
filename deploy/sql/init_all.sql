
-- 创建数据库
CREATE DATABASE IF NOT EXISTS `campushub_user` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `campushub_main` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `campushub_chat` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Demo 示例表（用于测试项目是否正常运行）
USE `campushub_main`;

CREATE TABLE IF NOT EXISTS `items` (
    `id` BIGINT NOT NULL COMMENT '主键ID',
    `name` VARCHAR(100) NOT NULL COMMENT '名称',
    `description` VARCHAR(500) DEFAULT '' COMMENT '描述',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1-正常 2-禁用',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` DATETIME DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Demo示例表';

INSERT INTO `items` (`id`, `name`, `description`, `status`) VALUES
(1, '测试项目1', '这是第一个测试项目', 1),
(2, '测试项目2', '这是第二个测试项目', 1),
(3, '测试项目3', '这是第三个测试项目（已禁用）', 2);

SELECT 'CampusHub 数据库初始化完成！' AS message;
