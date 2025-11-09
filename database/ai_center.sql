/*
 Navicat Premium Data Transfer

 Source Server         : localhost
 Source Server Type    : MySQL
 Source Server Version : 80041 (8.0.41)
 Source Host           : localhost:3306
 Source Schema         : ai_center

 Target Server Type    : MySQL
 Target Server Version : 80041 (8.0.41)
 File Encoding         : 65001

 Date: 25/08/2025 15:46:19
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for api_keys
-- ----------------------------
DROP TABLE IF EXISTS `api_keys`;
CREATE TABLE `api_keys`  (
  `id` int NOT NULL AUTO_INCREMENT,
  `api_key` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `permission` bigint NOT NULL,
  `is_active` tinyint(1) NOT NULL DEFAULT 1,
  `created_at` datetime(3) NULL DEFAULT NULL,
  `updated_at` datetime(3) NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `uni_api_keys_api_key`(`api_key` ASC) USING BTREE,
  INDEX `idx_api_key`(`api_key` ASC) USING BTREE,
  INDEX `idx_permission`(`permission` ASC) USING BTREE,
  INDEX `idx_is_active`(`is_active` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 28 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Table structure for group_chat_messages
-- ----------------------------
DROP TABLE IF EXISTS `group_chat_messages`;
CREATE TABLE `group_chat_messages`  (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `msg_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '消息ID',
  `msg_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '消息类型',
  `group_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '群组ID',
  `group_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '群组名称',
  `sender_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '发送者ID',
  `sender_role` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '发送者角色',
  `sender_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '发送者昵称',
  `weight` tinyint(1) NOT NULL DEFAULT 0 COMMENT '说话人排序权重，权重越大排序越前',
  `is_bot` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否机器人消息：0否，1是',
  `title` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL COMMENT '内容摘要',
  `content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL COMMENT '消息内容',
  `url` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL COMMENT '用户发送的URL链接',
  `message_time` datetime NOT NULL COMMENT '消息发送时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_group_id`(`group_id` ASC) USING BTREE,
  INDEX `idx_message_time`(`message_time` ASC) USING BTREE,
  INDEX `idx_sender_id`(`sender_id` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 7469 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '群聊天消息表' ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Table structure for group_chat_summaries
-- ----------------------------
DROP TABLE IF EXISTS `group_chat_summaries`;
CREATE TABLE `group_chat_summaries`  (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `group_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '群组ID',
  `group_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '群组名称',
  `summary_type` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'daily' COMMENT '总结类型：daily日报、weekly周报、monthly月报',
  `summary_content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '总结内容',
  `start_time` datetime NOT NULL COMMENT '总结开始时间',
  `end_time` datetime NOT NULL COMMENT '总结结束时间',
  `message_count` int NOT NULL DEFAULT 0 COMMENT '总结的消息数量',
  `participant_count` int NOT NULL DEFAULT 0 COMMENT '参与讨论的人数',
  `key_topics` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL COMMENT '关键话题',
  `topic_count` int NOT NULL DEFAULT 0 COMMENT '话题数',
  `hot_topics` json NULL COMMENT '热门话题问题，JSON格式存储',
  `active_hours` int NOT NULL DEFAULT 0 COMMENT '群总活跃时长（小时）',
  `active_periods` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '群活跃时间段，格式如07-08,10-12,15-16',
  `most_active_user` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '当天最活跃用户',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_group_id`(`group_id` ASC) USING BTREE,
  INDEX `idx_summary_type`(`summary_type` ASC) USING BTREE,
  INDEX `idx_start_time`(`start_time` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 89 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '群聊天总结表' ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Table structure for group_configs
-- ----------------------------
DROP TABLE IF EXISTS `group_configs`;
CREATE TABLE `group_configs`  (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `group_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '群组ID',
  `group_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '群组名称',
  `is_enabled` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否启用总结：0禁用，1启用',
  `summary_frequency` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'daily' COMMENT '总结频率：daily每日、weekly每周、monthly每月',
  `summary_time` time NULL DEFAULT '23:00:00' COMMENT '总结时间（每日总结的时间点）',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`) USING BTREE,
  UNIQUE INDEX `uk_group_id`(`group_id` ASC) USING BTREE,
  INDEX `idx_is_enabled`(`is_enabled` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 8 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '群配置表' ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Table structure for recommendation_cache
-- ----------------------------
DROP TABLE IF EXISTS `recommendation_cache`;
CREATE TABLE `recommendation_cache`  (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `cid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '用户ID',
  `recommendations` json NOT NULL COMMENT '推荐内容JSON',
  `user_profile` json NULL COMMENT '用户画像JSON',
  `algorithm` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'profile_based' COMMENT '推荐算法',
  `generated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '生成时间',
  `pushed` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否已推送',
  `pushed_at` datetime NULL DEFAULT NULL COMMENT '推送时间',
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_cid`(`cid` ASC) USING BTREE,
  INDEX `idx_generated_at`(`generated_at` ASC) USING BTREE,
  INDEX `idx_pushed`(`pushed` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 2905 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci COMMENT = '推荐内容缓存表' ROW_FORMAT = Dynamic;

-- ----------------------------
-- Table structure for session_messages
-- ----------------------------
DROP TABLE IF EXISTS `session_messages`;
CREATE TABLE `session_messages`  (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `api_key` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `session_id` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `role` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `message_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `seq` int NOT NULL,
  `user_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  `msg_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT '',
  `model_used` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL,
  PRIMARY KEY (`id`) USING BTREE,
  INDEX `idx_session_seq`(`session_id` ASC, `seq` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 840 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Table structure for session_summaries
-- ----------------------------
DROP TABLE IF EXISTS `session_summaries`;
CREATE TABLE `session_summaries`  (
  `session_id` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `session_title` varchar(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`session_id`) USING BTREE
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Table structure for simi_community_history
-- ----------------------------
DROP TABLE IF EXISTS `simi_community_history`;
CREATE TABLE `simi_community_history`  (
  `article_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `source_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `article_type` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `article_state` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `create_time` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `article_user_cid` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `article_text` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `images` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `label` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `score` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `law` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `law_image` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `politics` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `politics_image` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `ad` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `ad_image` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  `hot_search_title` text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Table structure for user_community_posting_record
-- ----------------------------
DROP TABLE IF EXISTS `user_community_posting_record`;
CREATE TABLE `user_community_posting_record`  (
  `cid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `time` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `behavior_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `article_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Table structure for user_join_exit_group_record
-- ----------------------------
DROP TABLE IF EXISTS `user_join_exit_group_record`;
CREATE TABLE `user_join_exit_group_record`  (
  `cid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `group_id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `time` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `operation_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `behavior_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  `group_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Table structure for user_profiles
-- ----------------------------
DROP TABLE IF EXISTS `user_profiles`;
CREATE TABLE `user_profiles`  (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `cid` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  `profile_json` json NOT NULL,
  `keywords` json NOT NULL,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`, `cid`) USING BTREE,
  UNIQUE INDEX `cid`(`cid` ASC) USING BTREE,
  INDEX `idx_updated_at`(`updated_at` ASC) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 3299 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = DYNAMIC;

SET FOREIGN_KEY_CHECKS = 1;
