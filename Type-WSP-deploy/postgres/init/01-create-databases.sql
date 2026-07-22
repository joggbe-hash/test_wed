-- ======================================================
-- PostgreSQL 初始化腳本
-- 在容器首次啟動時自動執行（掛載於 /docker-entrypoint-initdb.d）
-- 建立兩個資料庫：user_db（使用者資料）和 system_db（系統資料）
-- ======================================================

-- ===== user_db：使用者帳號相關 =====
CREATE DATABASE user_db;
GRANT ALL PRIVILEGES ON DATABASE user_db TO app_admin;

\connect user_db

-- 使用者資料表
CREATE TABLE users (
    id            SERIAL       PRIMARY KEY,              -- 自動遞增主鍵
    username      VARCHAR(50)  NOT NULL,                 -- 顯示名稱
    email         VARCHAR(255) NOT NULL UNIQUE,          -- 登入用信箱（唯一）
    password_hash VARCHAR(255) NOT NULL,                 -- bcrypt 雜湊後的密碼
    created_at    TIMESTAMP    NOT NULL DEFAULT NOW()    -- 建立時間
);

-- ===== system_db：貼文等系統資料 =====
-- system_db 是 docker-compose 中 POSTGRES_DB 指定的預設資料庫
\connect system_db

-- 貼文資料表
CREATE TABLE posts (
    id            SERIAL       PRIMARY KEY,                        -- 自動遞增主鍵
    user_id       INTEGER      NOT NULL,                           -- 發文者 ID（對應 user_db.users.id，跨 DB 不設 FK）
    username      VARCHAR(50)  NOT NULL,                           -- 發文者名稱（反正規化，避免跨 DB JOIN）
    content       TEXT,                                            -- 貼文文字內容
    image_url     TEXT,                                            -- 圖片路徑 JSON 陣列，如 ["processed/a.jpg","processed/b.jpg"]
    image_status  VARCHAR(20)  NOT NULL DEFAULT 'none',            -- 圖片處理狀態：none / processing / ready / failed
    created_at    TIMESTAMP    NOT NULL DEFAULT NOW()               -- 發文時間
);

-- 按時間倒序索引，加速 feed 查詢
CREATE INDEX idx_posts_feed_cursor ON posts (created_at DESC, id DESC);
