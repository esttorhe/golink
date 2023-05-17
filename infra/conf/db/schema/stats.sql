/*
 Navicat PostgreSQL Data Transfer

 Source Server         : golinks
 Source Server Type    : PostgreSQL
 Source Server Version : 150003 (150003)
 Source Host           : localhost:5432
 Source Catalog        : golinks
 Source Schema         : public

 Target Server Type    : PostgreSQL
 Target Server Version : 150003 (150003)
 File Encoding         : 65001

 Date: 15/05/2023 14:48:35
*/


-- ----------------------------
-- Table structure for stats
-- ----------------------------
CREATE TABLE "public"."stats" (
  "id" text COLLATE "pg_catalog"."default" NOT NULL,
  "clicks" int4,
  "created_at" timestamptz(6) DEFAULT now(),
  "updated_at" timestamptz(6) DEFAULT now(),
  "deleted_at" timestamptz(6),
  "created" timestamptz(6) DEFAULT now()
)
;
ALTER TABLE "public"."stats" OWNER TO "postgres";

-- ----------------------------
-- Primary Key structure for table stats
-- ----------------------------
ALTER TABLE "public"."stats" ADD CONSTRAINT "stats_pkey" PRIMARY KEY ("id");
