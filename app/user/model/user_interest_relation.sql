CREATE TABLE `user_interest_relation` (
                                          `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID，关联user表的user_id（命名统一）',
                                          `tag_id` INT UNSIGNED NOT NULL COMMENT '标签ID，关联interest_tag的tag_id',
                                          `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '关联创建时间',
                                          PRIMARY KEY (`user_id`, `tag_id`), -- 联合主键，保证一个用户不会重复绑定同一个标签
    -- 外键约束适配用户表的user_id字段（逻辑无变化，仅注释更清晰）
                                          FOREIGN KEY (`user_id`) REFERENCES `user`(`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
                                          FOREIGN KEY (`tag_id`) REFERENCES `interest_tag`(`tag_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户-兴趣标签关联表';