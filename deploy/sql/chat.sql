-- ============================================
-- 聊天服务数据库
-- Database: campushub_chat
-- ============================================
-- 负责人：马华恩
-- 包含：群聊表、群成员表、消息表、通知表

CREATE DATABASE IF NOT EXISTS `campushub_chat`
    DEFAULT CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE `campushub_chat`;

-- ============================================
-- 群聊表 (groups)
-- ============================================
CREATE TABLE IF NOT EXISTS `groups` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增主键',
    `group_id` VARCHAR(64) NOT NULL COMMENT '群聊唯一标识',
    `activity_id` VARCHAR(64) NOT NULL COMMENT '关联活动ID',
    `name` VARCHAR(255) NOT NULL COMMENT '群聊名称',
    `owner_id` VARCHAR(64) NOT NULL COMMENT '群主用户ID',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1-正常 2-已解散',
    `max_members` INT NOT NULL COMMENT '最大成员数',
    `member_count` INT NOT NULL DEFAULT 0 COMMENT '当前成员数量',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME DEFAULT NULL COMMENT '删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_group_id` (`group_id`),
    UNIQUE KEY `uk_activity_id` (`activity_id`),
    KEY `idx_owner_id` (`owner_id`),
    KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='群聊表';

-- ============================================
-- 群成员表 (group_members)
-- ============================================
CREATE TABLE IF NOT EXISTS `group_members` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增主键',
    `group_id` VARCHAR(64) NOT NULL COMMENT '群聊ID',
    `user_id` VARCHAR(64) NOT NULL COMMENT '用户ID',
    `role` TINYINT NOT NULL DEFAULT 1 COMMENT '角色: 1-普通成员 2-群主',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1-正常 2-已退出',
    `joined_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '加入时间',
    `left_at` DATETIME DEFAULT NULL COMMENT '退出时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_group_user` (`group_id`, `user_id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='群成员表';

-- ============================================
-- 消息表 (messages)
-- ============================================
CREATE TABLE IF NOT EXISTS `messages` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '消息ID（自增主键）',
    `message_id` VARCHAR(64) NOT NULL COMMENT '消息唯一标识',
    `group_id` VARCHAR(64) NOT NULL COMMENT '群聊ID',
    `sender_id` VARCHAR(64) NOT NULL COMMENT '发送者用户ID',
    `msg_type` TINYINT NOT NULL COMMENT '消息类型: 1-文字 2-图片',
    `content` TEXT COMMENT '文本内容',
    `image_url` VARCHAR(512) COMMENT '图片URL',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1-正常 2-已撤回',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_message_id` (`message_id`),
    KEY `idx_group_id_created` (`group_id`, `created_at`),
    KEY `idx_sender_id` (`sender_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='消息表';

-- ============================================
-- 通知表 (notifications)
-- ============================================
CREATE TABLE IF NOT EXISTS `notifications` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增主键',
    `notification_id` VARCHAR(64) NOT NULL COMMENT '通知唯一标识',
    `user_id` VARCHAR(64) NOT NULL COMMENT '用户ID',
    `type` VARCHAR(32) NOT NULL COMMENT '通知类型: system-系统通知, group_invite-群邀请等',
    `title` VARCHAR(255) NOT NULL COMMENT '通知标题',
    `content` TEXT NOT NULL COMMENT '通知内容',
    `data` JSON COMMENT '附加数据（JSON格式）',
    `is_read` TINYINT NOT NULL DEFAULT 0 COMMENT '是否已读: 0-未读 1-已读',
    `read_at` DATETIME DEFAULT NULL COMMENT '阅读时间',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_notification_id` (`notification_id`),
    KEY `idx_user_id_created` (`user_id`, `created_at`),
    KEY `idx_is_read` (`is_read`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通知表';
