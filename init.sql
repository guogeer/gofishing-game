-- MySQL 8

CREATE DATABASE IF NOT EXISTS game;
USE game;

DROP TABLE IF EXISTS user_info;
CREATE TABLE user_info (
    id INT AUTO_INCREMENT PRIMARY KEY,
    nickname VARCHAR(50) NOT NULL,
    sex int not null,
    icon varchar(255) not null,
    server_name varchar(32) not null,
    time_zone float not null,
    email varchar(64) not null,
    ip varchar(32) not null,
    client_version varchar(32) not null,
    mac varchar(24) not null,
    imei varchar(24) not null,
    imsi varchar(24) not null,
    create_time TIMESTAMPS not null default current_timestamps
);

DROP TABLE IF EXISTS item_log;
CREATE TABLE item_log (
    id INT AUTO_INCREMENT PRIMARY KEY,
	`uid` INT NOT NULL,
    item_id INT NOT NULL,
	num INT NOT NULL,
	balance INT NOT NULL,
    extra JSON NOT NULL,
	uuid varchar(64) NOT NULL,
    create_time TIMESTAMPS NOT NULL
);

DROP TABLE IF EXISTS online_log;
CREATE TABLE online_log (
    id INT AUTO_INCREMENT PRIMARY KEY,
	`uid` INT NOT NULL,
    ip varchar(48) NOT NULL,
	imei varchar(18) NOT NULL,
	imsi varchar(16) NOT NULL,
    chan_id varchar(32) NOT NULL,
	client_version varchar(32) NOT NULL,
    login_time TIMESTAMPS NOT NULL,
	offline_time TIMESTAMPS,
);

DROP TABLE IF EXISTS user_plate;
CREATE TABLE user_plate (
    id INT AUTO_INCREMENT PRIMARY KEY,
	`uid` INT NOT NULL,
    plate varchar(16) not null,
    open_id varchar(48) NOT NULL,
	create_time TIMESTAMPS NOT NULL,
    index idx_uid(`uid`),
    unique index idx_open_id(open_id)
);

DROP TABLE IF EXISTS user_bin;
CREATE TABLE user_bin (
    id INT AUTO_INCREMENT PRIMARY KEY,
	`uid` INT NOT NULL,
    `class` varchar(16) not null,
    bin blob not null,
	create_time TIMESTAMPS NOT NULL,
    unique index idx_uid_class(`uid`,`class`)
);

select id,`type`,recv_uid,`data`,`status`,send_time

DROP TABLE IF EXISTS mail;
CREATE TABLE mail (
    id INT AUTO_INCREMENT PRIMARY KEY,
	`type` INT NOT NULL,
    send_uid int not null,
    recv_uid int not null,
    `status` int not null,
    `data` text not null,
	send_time TIMESTAMPS NOT NULL default current_timestamps,
    index idx_recv_uid(recv_uid)
);

DROP TABLE IF EXISTS dict;
CREATE TABLE dict (
    id INT AUTO_INCREMENT PRIMARY KEY,
    `key` varchar(32) not null,
    `value` JSON not null,
	update_time TIMESTAMPS NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    index idx_key(`key`)
);

CREATE DATABASE IF NOT EXISTS manage;
USE manage;

DROP TABLE IF EXISTS gm_table;
CREATE TABLE gm_table (
    id INT AUTO_INCREMENT PRIMARY KEY,
    `name` varchar(32) not null,
	`version` int NOT NULL,
    content text not null,
	update_time TIMESTAMPS NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    index idx_name(`name`)
);

DROP TABLE IF EXISTS gm_script;
CREATE TABLE gm_script (
    id INT AUTO_INCREMENT PRIMARY KEY,
    `name` varchar(32) not null,
    body text not null,
	update_time TIMESTAMPS NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    index idx_name(`name`)
);