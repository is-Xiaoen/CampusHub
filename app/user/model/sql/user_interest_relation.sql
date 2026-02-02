CREATE TABLE `user_interest_relations` (
                                        id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
                                        user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
                                        tag_id BIGINT UNSIGNED NOT NULL DEFAULT 0,  -- 关键修改：interest_id → tag_id
                                        create_time datetime DEFAULT CURRENT_TIMESTAMP,
                                        PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;