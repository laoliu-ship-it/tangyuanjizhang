CREATE DATABASE IF NOT EXISTS fandian CHARACTER SET utf8mb4;
USE fandian;

CREATE TABLE users (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    username      VARCHAR(50) UNIQUE NOT NULL,
    email         VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at    DATETIME NULL
);

CREATE TABLE tenants (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    name          VARCHAR(100) NOT NULL,
    owner_id      BIGINT NOT NULL,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at    DATETIME NULL,
    FOREIGN KEY (owner_id) REFERENCES users(id)
);

CREATE TABLE tenant_members (
    tenant_id     BIGINT NOT NULL,
    user_id       BIGINT NOT NULL,
    role          ENUM('admin','editor','viewer','partner','finance') DEFAULT 'viewer',
    joined_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, user_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (user_id)   REFERENCES users(id)
);

CREATE TABLE categories (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    tenant_id     BIGINT NOT NULL,
    name          VARCHAR(50) NOT NULL,
    type          ENUM('income','expense') NOT NULL,
    icon          VARCHAR(50) DEFAULT '',
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at    DATETIME NULL,
    UNIQUE KEY uk_tenant_name_type (tenant_id, name, type),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE merchants (
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    tenant_id     BIGINT UNSIGNED NOT NULL,
    name          VARCHAR(100) NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at    DATETIME NULL,
    INDEX idx_tenant_id (tenant_id),
    INDEX idx_deleted_at (deleted_at)
);

CREATE TABLE transactions (
    id                BIGINT AUTO_INCREMENT PRIMARY KEY,
    tenant_id         BIGINT NOT NULL,
    user_id           BIGINT NOT NULL,
    type              ENUM('income','expense') NOT NULL,
    amount            DECIMAL(12,2) NOT NULL,
    category_id       BIGINT NOT NULL,
    merchant_id       BIGINT UNSIGNED NOT NULL DEFAULT 0,
    transaction_date  DATETIME NOT NULL,
    note              VARCHAR(255) DEFAULT '',
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at        DATETIME NULL,
    FOREIGN KEY (tenant_id)   REFERENCES tenants(id),
    FOREIGN KEY (user_id)     REFERENCES users(id),
    FOREIGN KEY (category_id) REFERENCES categories(id),
    INDEX idx_tenant_date (tenant_id, transaction_date)
);

CREATE TABLE transaction_images (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY,
    transaction_id  BIGINT NOT NULL,
    image_path      VARCHAR(500) NOT NULL,
    ocr_amount      DECIMAL(12,2) NOT NULL DEFAULT 0,
    ocr_date        VARCHAR(20) NOT NULL DEFAULT '',
    ocr_merchant    VARCHAR(100) NOT NULL DEFAULT '',
    ocr_raw_texts   TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE
);
