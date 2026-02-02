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
CREATE TABLE `user` (
    `user_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户主键ID（与关联表user_id完全统一）',
    `QQemail` VARCHAR(100) NOT NULL COMMENT 'QQ邮箱（用户登录/标识用）',
    `nickname` VARCHAR(50) NOT NULL COMMENT '用户昵称',
    `avatar_url` VARCHAR(255) DEFAULT '' COMMENT '用户头像URL地址',
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

-- TODO(杨春路): user_details 表字段待补充（nickname, avatar, phone 等）
-- TODO(杨春路): user_tags 用户兴趣标签表

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
CREATE TABLE `student_verifications` (
    `id` bigint NOT NULL COMMENT '主键ID',
    `user_id` bigint NOT NULL COMMENT '用户ID',
    `status` tinyint NOT NULL DEFAULT 0 COMMENT '认证状态：0未认证 1待审核 2已认证 3已拒绝',
    `real_name` varchar(50) COMMENT '真实姓名',
    `school_name` varchar(100) COMMENT '学校名称',
    `student_id` varchar(50) COMMENT '学号',
    `department` varchar(100) COMMENT '院系',
    `admission_year` varchar(10) COMMENT '入学年份',
    `reject_reason` varchar(255) COMMENT '拒绝原因',
    `verified_at` datetime COMMENT '认证通过时间',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_user_id` (`user_id`),
    UNIQUE KEY `uk_student_id` (`student_id`)
) ENGINE=InnoDB COMMENT='学生认证表';


CREATE TABLE `interest_tag` (
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


CREATE TABLE user_interest_relation (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
    tag_id BIGINT UNSIGNED NOT NULL DEFAULT 0,  -- 关键修改：interest_id → tag_id
    create_time datetime DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
