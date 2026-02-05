-- ============================================
-- 活动服务数据库
-- Database: campushub_main
-- ============================================
-- 负责人：Xiaoen、谢玉林
-- TODO: 根据需求设计表结构

CREATE DATABASE IF NOT EXISTS `campushub_main`
    DEFAULT CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE `campushub_main`;

-- ============================================
-- Xiaoen 负责的表（活动核心模块）
-- ============================================

-- 1. categories 活动分类表
CREATE TABLE IF NOT EXISTS `categories` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '分类ID',
    `name` VARCHAR(50) NOT NULL COMMENT '分类名称',
    `icon` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '分类图标(FontAwesome类名)',
    `sort` INT NOT NULL DEFAULT 0 COMMENT '排序权重(越大越靠前)',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1启用 0禁用',
    `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
    `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COMMENT='活动分类表';

-- 分类初始数据
INSERT INTO `categories` (`id`, `name`, `icon`, `sort`, `status`, `created_at`, `updated_at`) VALUES
(1, '学术讲座', 'fa-graduation-cap', 100, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
(2, '社团活动', 'fa-users', 90, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
(3, '体育运动', 'fa-futbol', 80, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
(4, '文艺演出', 'fa-music', 70, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
(5, '志愿服务', 'fa-heart', 60, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
(6, '比赛竞技', 'fa-trophy', 50, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
(7, '招聘就业', 'fa-briefcase', 40, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
(8, '其他', 'fa-ellipsis-h', 0, 1, UNIX_TIMESTAMP(), UNIX_TIMESTAMP());

-- 2. activities 活动主表
CREATE TABLE IF NOT EXISTS `activities` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '活动ID',

    -- 基本信息
    `title` VARCHAR(100) NOT NULL COMMENT '活动标题',
    `cover_url` VARCHAR(500) NOT NULL COMMENT '封面URL',
    `cover_type` TINYINT NOT NULL DEFAULT 1 COMMENT '封面类型: 1图片 2视频',
    `description` TEXT COMMENT '活动详情(富文本)',
    `category_id` BIGINT UNSIGNED NOT NULL COMMENT '分类ID',

    -- 组织者信息（冗余存储，避免联表查询）
    `organizer_id` BIGINT UNSIGNED NOT NULL COMMENT '组织者用户ID',
    `organizer_name` VARCHAR(50) NOT NULL COMMENT '组织者名称',
    `organizer_avatar` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '组织者头像',
    `contact_phone` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '联系电话',

    -- 时间信息（Unix 时间戳，秒）
    `register_start_time` BIGINT NOT NULL COMMENT '报名开始时间',
    `register_end_time` BIGINT NOT NULL COMMENT '报名截止时间',
    `activity_start_time` BIGINT NOT NULL COMMENT '活动开始时间',
    `activity_end_time` BIGINT NOT NULL COMMENT '活动结束时间',

    -- 地点信息
    `location` VARCHAR(200) NOT NULL COMMENT '活动地点',
    `address_detail` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '详细地址',
    `longitude` DECIMAL(10,7) DEFAULT NULL COMMENT '经度',
    `latitude` DECIMAL(10,7) DEFAULT NULL COMMENT '纬度',

    -- 名额与报名规则
    `max_participants` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最大参与人数(0=不限)',
    `current_participants` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '当前报名人数',
    `require_approval` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否需要审批',
    `require_student_verify` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否需要学生认证',
    `min_credit_score` INT NOT NULL DEFAULT 0 COMMENT '最低信用分要求',

    -- 状态
    `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态: 0草稿 1待审核 2已发布 3进行中 4已结束 5已拒绝 6已取消',
    `reject_reason` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '拒绝/取消原因',

    -- 统计（异步更新）
    `view_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '浏览量',
    `like_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '点赞数',

    -- 乐观锁
    `version` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '乐观锁版本号',

    -- 时间戳
    `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
    `updated_at` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间',
    `deleted_at` DATETIME DEFAULT NULL COMMENT '软删除时间',

    PRIMARY KEY (`id`),
    KEY `idx_category_status` (`category_id`, `status`),
    KEY `idx_status_start` (`status`, `activity_start_time`),
    KEY `idx_organizer` (`organizer_id`),
    KEY `idx_created_at` (`created_at`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COMMENT='活动表';

-- 3. activity_tags 活动-标签关联表
CREATE TABLE IF NOT EXISTS `activity_tags` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '关联ID',
    `activity_id` BIGINT UNSIGNED NOT NULL COMMENT '活动ID',
    `tag_id` BIGINT UNSIGNED NOT NULL COMMENT '标签ID',
    `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_activity_tag` (`activity_id`, `tag_id`),
    KEY `idx_tag_id` (`tag_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COMMENT='活动-标签关联表';

-- 4. activity_status_logs 状态变更日志表
CREATE TABLE IF NOT EXISTS `activity_status_logs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '日志ID',
    `activity_id` BIGINT UNSIGNED NOT NULL COMMENT '活动ID',
    `from_status` TINYINT NOT NULL COMMENT '变更前状态',
    `to_status` TINYINT NOT NULL COMMENT '变更后状态',
    `operator_id` BIGINT UNSIGNED NOT NULL COMMENT '操作人ID(0=系统)',
    `operator_type` TINYINT NOT NULL DEFAULT 1 COMMENT '操作人类型: 1用户 2管理员 3系统',
    `reason` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '变更原因',
    `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
    PRIMARY KEY (`id`),
    KEY `idx_activity_created` (`activity_id`, `created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COMMENT='活动状态变更日志表';

-- 5. tag_cache 标签缓存表（从用户服务同步）
CREATE TABLE IF NOT EXISTS `tag_cache` (
    `id` BIGINT UNSIGNED NOT NULL COMMENT '标签ID（与用户服务一致）',
    `name` VARCHAR(50) NOT NULL COMMENT '标签名称',
    `color` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '标签颜色（如 #FF6B6B）',
    `icon` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '标签图标URL',
    `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态: 1启用 0禁用',
    `description` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '标签描述',
    `synced_at` BIGINT NOT NULL COMMENT '最后同步时间戳',
    `created_at` BIGINT NOT NULL COMMENT '原始创建时间戳',
    `updated_at` BIGINT NOT NULL COMMENT '原始更新时间戳',
    PRIMARY KEY (`id`),
    KEY `idx_status` (`status`),
    KEY `idx_synced_at` (`synced_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='标签缓存表（从用户服务同步）';

-- 6. activity_tag_stats 标签统计表
CREATE TABLE IF NOT EXISTS `activity_tag_stats` (
    `tag_id` BIGINT UNSIGNED NOT NULL COMMENT '标签ID',
    `activity_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关联的活动数量',
    `view_count` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '标签关联活动的总浏览量',
    `updated_at` BIGINT NOT NULL COMMENT '更新时间戳',
    PRIMARY KEY (`tag_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='活动标签统计表';

-- ============================================
-- 谢玉林负责的表（票券模块）
-- ============================================

-- 7. activity_tickets 票据表
CREATE TABLE IF NOT EXISTS `activity_tickets` (
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

-- 8. check_in_records 核销记录表（幂等保障）
CREATE TABLE IF NOT EXISTS `check_in_records` (
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

-- 9. activity_registrations 报名记录表
CREATE TABLE IF NOT EXISTS `activity_registrations` (
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

-- ============================================
-- DTM 分布式事务相关表
-- ============================================

-- 10. dtm_barrier DTM 子事务屏障表
-- 用于解决分布式事务的三大问题：幂等、空补偿、悬挂
-- 参考：https://en.dtm.pub/practice/barrier.html
CREATE TABLE IF NOT EXISTS `dtm_barrier` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `trans_type` VARCHAR(45) NOT NULL DEFAULT '' COMMENT '事务类型（saga/tcc/xa）',
    `gid` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '全局事务ID',
    `branch_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '分支事务ID',
    `op` VARCHAR(45) NOT NULL DEFAULT '' COMMENT '操作类型（action/compensate/try/confirm/cancel）',
    `barrier_id` VARCHAR(45) NOT NULL DEFAULT '' COMMENT '屏障ID',
    `reason` VARCHAR(45) NOT NULL DEFAULT '' COMMENT '插入原因（committed/rollbacked）',
    `create_time` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_gid_branchid_op_barrierid` (`gid`, `branch_id`, `op`, `barrier_id`),
    KEY `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=DYNAMIC COMMENT='DTM子事务屏障表';
