-- hzw.sso 开发环境 MySQL 初始化脚本
-- 由 docker-compose 在首次启动时自动执行

CREATE DATABASE IF NOT EXISTS `sso` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `hydra` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
