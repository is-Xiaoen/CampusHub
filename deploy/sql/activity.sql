-- ============================================
-- 活动服务数据库
-- Database: campushub_main
-- ============================================
-- 负责人：马肖阳、谢玉林
-- TODO: 根据需求设计表结构

CREATE DATABASE IF NOT EXISTS `campushub_main`
    DEFAULT CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE `campushub_main`;

-- TODO(马肖阳): categories 活动分类表
-- TODO(马肖阳): activities 活动表
-- TODO(马肖阳): activity_tags 活动标签表

-- 1. activity_tickets 票据表
CREATE TABLE `activity_tickets` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '票据主键ID',
    `ticket_code` varchar(32) NOT NULL COMMENT '票据短码',
    `ticket_uuid` char(36) NOT NULL COMMENT '票据UUID',
    `activity_id` bigint NOT NULL COMMENT '活动ID',
    `user_id` bigint NOT NULL COMMENT '持票用户ID',
    `registration_id` bigint NOT NULL COMMENT '关联报名记录ID',
    `totp_secret` varchar(64) COMMENT 'TOTP密钥',
    `totp_enabled` tinyint NOT NULL DEFAULT 1 COMMENT '是否启用TOTP',
    `valid_start_time` bigint NOT NULL DEFAULT 0 COMMENT '可核销开始时间',
    `valid_end_time` bigint NOT NULL DEFAULT 0 COMMENT '可核销截止时间',
    `status` tinyint NOT NULL DEFAULT 0 COMMENT '状态:0未使用 1已使用 2已过期 3已作废',
    `used_time` bigint NOT NULL DEFAULT 0 COMMENT '核销时间',
    `used_location` varchar(200) DEFAULT '' COMMENT '核销地点',
    `check_in_snapshot` json COMMENT '核销快照',
    `created_at` bigint NOT NULL DEFAULT 0 COMMENT '创建时间',
    `updated_at` bigint NOT NULL DEFAULT 0 COMMENT '更新时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_ticket_code` (`ticket_code`),
    UNIQUE KEY `uk_ticket_uuid` (`ticket_uuid`),
    UNIQUE KEY `uk_registration_id` (`registration_id`),
    KEY `idx_activity_id` (`activity_id`),
    KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 COMMENT='活动票据表';

-- 2. check_in_records 核销记录表（幂等保障）
CREATE TABLE `check_in_records` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '核销记录ID',
    `check_in_no` varchar(64) NOT NULL COMMENT '核销流水号',
    `ticket_id` bigint NOT NULL COMMENT '票据ID',
    `ticket_code` varchar(32) NOT NULL COMMENT '票据短码',
    `activity_id` bigint NOT NULL COMMENT '活动ID',
    `user_id` bigint NOT NULL COMMENT '用户ID',
    `check_in_time` bigint NOT NULL DEFAULT 0 COMMENT '核销时间',
    `longitude` decimal(10,7) DEFAULT NULL COMMENT '经度',
    `latitude` decimal(10,7) DEFAULT NULL COMMENT '纬度',
    `client_request_id` varchar(64) NOT NULL COMMENT '请求ID(幂等)',
    `check_in_snapshot` text COMMENT '核销快照(JSON字符串)',
    `created_at` bigint NOT NULL DEFAULT 0 COMMENT '创建时间',
    `deleted_at` datetime DEFAULT NULL COMMENT '删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_check_in_no` (`check_in_no`),
    UNIQUE KEY `uk_client_request_id` (`client_request_id`),
    KEY `idx_ticket_id` (`ticket_id`),
    KEY `idx_activity_id` (`activity_id`),
    KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 COMMENT='核销记录表';

-- 3. activity_registrations 报名记录表
CREATE TABLE `activity_registrations` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '报名ID',
    `activity_id` bigint NOT NULL COMMENT '活动ID',
    `user_id` bigint NOT NULL COMMENT '用户ID',
    `status` tinyint NOT NULL DEFAULT 1 COMMENT '报名状态: 1成功 2取消 3失败',
    `cancel_time` bigint NOT NULL DEFAULT 0 COMMENT '取消时间',
    `created_at` bigint NOT NULL DEFAULT 0 COMMENT '报名时间',
    `updated_at` bigint NOT NULL DEFAULT 0 COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_activity_user` (`activity_id`, `user_id`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_activity_id` (`activity_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 COMMENT='活动报名记录表';
