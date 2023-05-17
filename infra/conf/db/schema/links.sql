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

 Date: 15/05/2023 14:48:25
*/


-- ----------------------------
-- Table structure for links
-- ----------------------------
CREATE TABLE "public"."links" (
  "id" text COLLATE "pg_catalog"."default" NOT NULL,
  "short" text COLLATE "pg_catalog"."default" NOT NULL,
  "long" text COLLATE "pg_catalog"."default" NOT NULL,
  "created" timestamptz(6) DEFAULT now(),
  "lastedit" timestamp(6),
  "owner" text COLLATE "pg_catalog"."default" NOT NULL,
  "created_at" timestamptz(6) DEFAULT now(),
  "updated_at" timestamptz(6) DEFAULT now(),
  "deleted_at" timestamptz(6),
  "last_edit" timestamptz(6)
)
;
ALTER TABLE "public"."links" OWNER TO "postgres";

-- ----------------------------
-- Indexes structure for table links
-- ----------------------------
CREATE INDEX "idx_links_deleted_at" ON "public"."links" USING btree (
  "deleted_at" "pg_catalog"."timestamptz_ops" ASC NULLS LAST
);

-- ----------------------------
-- Primary Key structure for table links
-- ----------------------------
ALTER TABLE "public"."links" ADD CONSTRAINT "id" PRIMARY KEY ("id");
