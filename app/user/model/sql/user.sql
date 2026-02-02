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