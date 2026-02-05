-- ============================================
-- DTM 分布式事务管理器数据库
-- Database: dtm
-- ============================================
-- 说明：DTM Server 启动时会自动创建所需的表
-- 此文件仅用于创建数据库

CREATE DATABASE IF NOT EXISTS `dtm`
    DEFAULT CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

-- DTM Server 会自动创建以下表：
-- 1. dtm_trans - 全局事务记录表
-- 2. dtm_branch - 分支事务记录表
-- 3. dtm_kv - 键值存储表（用于工作流等）
--
-- 详细说明见：https://en.dtm.pub/ref/mysql.html
