CREATE TABLE `interest_tag` (
                                `tag_id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '标签主键ID，自增',
                                `tag_name` VARCHAR(50) NOT NULL COMMENT '标签名称（如：运动、音乐、阅读）',
                                `tag_desc` VARCHAR(200) DEFAULT '' COMMENT '标签描述（可选，如：运动包含跑步、篮球等）',
                                `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '标签创建时间',
                                `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '标签更新时间',
                                PRIMARY KEY (`tag_id`),
                                UNIQUE KEY `uk_tag_name` (`tag_name`) -- 保证标签名称唯一，避免重复
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='兴趣标签基础表';