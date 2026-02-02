CREATE TABLE user_interest_relation (
                                        id bigint NOT NULL AUTO_INCREMENT,
                                        user_id bigint NOT NULL DEFAULT 0,
                                        interest_id bigint NOT NULL DEFAULT 0,
                                        create_time datetime DEFAULT CURRENT_TIMESTAMP,
                                        PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;