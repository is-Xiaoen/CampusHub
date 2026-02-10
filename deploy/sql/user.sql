-- ============================================
-- 用户服务数据库
-- Database: campushub_user
-- ============================================
-- 负责人：杨春路、王得贤
-- TODO: 根据需求设计表结构

CREATE DATABASE IF NOT EXISTS `campushub_user`
    DEFAULT CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE `campushub_user`;

-- 0. user_details 用户表（壳子，具体字段由杨春路补充）
CREATE TABLE `users` (
    `user_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户主键ID（与关联表user_id完全统一）',
    `QQemail` VARCHAR(100) NOT NULL COMMENT 'QQ邮箱（用户登录/标识用）',
    `nickname` VARCHAR(50) NOT NULL COMMENT '用户昵称',
    `avatar_url` VARCHAR(255) DEFAULT '' COMMENT '用户头像URL地址',
    `avatar_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '用户头像图片ID，关联sys_images表',
    `introduction` VARCHAR(500) DEFAULT '' COMMENT '用户个人简介',
    `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '用户状态：0-禁用，1-正常，2-注销',
    `password` VARCHAR(255) NOT NULL COMMENT '用户密码（建议存储加密后的值，如BCrypt哈希）',
    `gender` TINYINT UNSIGNED DEFAULT 0 COMMENT '性别：0-未知，1-男，2-女',
    `age` TINYINT UNSIGNED DEFAULT 0 COMMENT '用户年龄（0表示未填写）',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '用户创建时间',
    `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '用户信息更新时间',
    PRIMARY KEY (`user_id`),
    UNIQUE KEY `uk_qqemail` (`QQemail`) -- 保证QQ邮箱唯一，避免重复注册
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户基础信息表';



-- 1. user_credits 信用分表（CreditService依赖）
CREATE TABLE `user_credits` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID（自增）',
    `user_id` bigint NOT NULL COMMENT '用户ID',
    `score` int NOT NULL DEFAULT 100 COMMENT '信用分数（0-100）',
    `level` tinyint NOT NULL DEFAULT 4 COMMENT '信用等级：0黑名单 1风险 2良好 3优秀 4社区之星',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 COMMENT='用户信用分表';

-- 2. credit_logs 信用变更记录表（幂等控制）
CREATE TABLE `credit_logs` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID（自增）',
    `user_id` bigint NOT NULL COMMENT '用户ID',
    `change_type` tinyint NOT NULL COMMENT '变更类型：1加分 2扣分',
    `source_id` varchar(128) NOT NULL COMMENT '来源ID（幂等键，如：init:10001, checkin:123:456）',
    `before_score` int NOT NULL COMMENT '变更前分数',
    `after_score` int NOT NULL COMMENT '变更后分数',
    `delta` int NOT NULL COMMENT '变更值（正数加分，负数扣分）',
    `reason` varchar(255) COMMENT '变更原因',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_source_id` (`source_id`),
    KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 COMMENT='信用变更记录表';

-- 3. student_verifications 学生认证表（VerifyService依赖）
-- 状态机：0初始 1OCR中 2待确认 3人工审核 4通过 5拒绝 6超时 7取消 8OCR失败
CREATE TABLE `student_verifications` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `user_id` bigint NOT NULL COMMENT '用户ID',
    `status` tinyint NOT NULL DEFAULT 0 COMMENT '认证状态：0初始 1OCR中 2待确认 3人工审核 4通过 5拒绝 6超时 7取消 8OCR失败',
    `real_name` varchar(100) COMMENT '真实姓名（AES加密）',
    `school_name` varchar(100) COMMENT '学校名称',
    `student_id` varchar(100) COMMENT '学号（AES加密）',
    `department` varchar(100) COMMENT '院系',
    `admission_year` varchar(10) COMMENT '入学年份',
    `front_image_url` varchar(500) DEFAULT '' COMMENT '学生证正面图片URL',
    `back_image_url` varchar(500) DEFAULT '' COMMENT '学生证详情面图片URL',
    `ocr_platform` varchar(20) NOT NULL DEFAULT '' COMMENT 'OCR平台：tencent/aliyun',
    `ocr_raw_json` text COMMENT 'OCR原始响应JSON（用于审计追溯）',
    `ocr_confidence` decimal(5,2) COMMENT 'OCR识别置信度（0-100）',
    `reject_reason` varchar(255) COMMENT '拒绝原因',
    `cancel_reason` varchar(255) COMMENT '取消原因',
    `reviewer_id` bigint COMMENT '审核人ID（人工审核时）',
    `operator` varchar(50) COMMENT '操作来源：user_apply/ocr_callback/manual_review/timeout_job',
    `verified_at` datetime COMMENT '认证通过时间',
    `ocr_completed_at` datetime COMMENT 'OCR完成时间',
    `reviewed_at` datetime COMMENT '人工审核时间',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_id` (`user_id`),
    UNIQUE KEY `uk_student_id` (`student_id`)
) ENGINE=InnoDB COMMENT='学生认证表';


CREATE TABLE `interest_tags` (
    `tag_id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '标签主键ID，自增',
    `tag_name` VARCHAR(50) NOT NULL COMMENT '标签名称（如：运动、音乐、阅读）',
    `color` VARCHAR(20) DEFAULT '' COMMENT '标签颜色值（如十六进制#FFFFFF、RGB值等）',
    `icon` VARCHAR(255) DEFAULT '' COMMENT '标签图标线上URL地址',
    `usage_count` INT UNSIGNED DEFAULT 0 COMMENT '标签被用户使用的总次数',
    `status` TINYINT UNSIGNED DEFAULT 1 COMMENT '标签状态：0-禁用，1-正常',
    `tag_desc` VARCHAR(200) DEFAULT '' COMMENT '标签描述（可选，如：运动包含跑步、篮球等）',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '标签创建时间',
    `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '标签更新时间',
    PRIMARY KEY (`tag_id`),
    UNIQUE KEY `uk_tag_name` (`tag_name`) -- 保证标签名称唯一，避免重复
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='兴趣标签基础表';


CREATE TABLE `user_interest_relations` (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    tag_id BIGINT UNSIGNED NOT NULL DEFAULT 0,  -- 关键修改：interest_id → tag_id
    create_time datetime DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================
-- DTM 分布式事务支持表
-- ============================================

-- dtm_barrier DTM 子事务屏障表
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


CREATE TABLE `sys_images` (
                              `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键，自增ID',
                              `url` varchar(500) NOT NULL COMMENT '图片存储相对路径或完整URL',
                              `origin_name` varchar(255) DEFAULT NULL COMMENT '原始文件名',
                              `biz_type` varchar(32) NOT NULL COMMENT '业务类型: avatar, activity_cover, identity_auth',
                              `file_size` int NOT NULL DEFAULT '0' COMMENT '文件大小(字节)',
                              `mime_type` varchar(64) DEFAULT NULL COMMENT '图片格式: image/jpeg, image/png等',
                              `extension` varchar(10) DEFAULT NULL COMMENT '后缀名: jpg, png',
                              `ref_count` int NOT NULL DEFAULT '0' COMMENT '核心字段：引用计数，默认为0',
                              `uploader_id` bigint NOT NULL COMMENT '上传者用户ID，用于权限校验',
                              `status` tinyint NOT NULL DEFAULT '0' COMMENT '状态: 0-审核中, 1-正常, 2-封禁, 3-待清理',
                              `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '上传时间',
                              `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后更新时间',
                              PRIMARY KEY (`id`),
    -- 索引优化
                              INDEX `idx_uploader` (`uploader_id`), -- 关键：用于实现你要求的身份校验逻辑
                              INDEX `idx_biz_status` (`biz_type`, `status`), -- 方便管理后台按业务和状态筛选
                              INDEX `idx_ref_count` (`ref_count`) -- 方便定时清理脚本扫描引用为0的孤儿图片
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='图片资源中心表';