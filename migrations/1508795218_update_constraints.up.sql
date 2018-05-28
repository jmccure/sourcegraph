
ALTER TABLE "comments"
	ALTER COLUMN "created_at" SET NOT NULL,
	ALTER COLUMN "updated_at" SET NOT NULL;

ALTER TABLE "threads"
	ALTER COLUMN "created_at" SET NOT NULL,
	ALTER COLUMN "updated_at" SET NOT NULL;

ALTER TABLE "org_members"
	ALTER COLUMN "created_at" SET NOT NULL,
	ALTER COLUMN "updated_at" SET NOT NULL;

ALTER TABLE "org_repos"
	ALTER COLUMN "created_at" SET NOT NULL,
	ALTER COLUMN "updated_at" SET NOT NULL;

ALTER TABLE "org_tags"
	ALTER COLUMN "created_at" SET NOT NULL,
	ALTER COLUMN "updated_at" SET NOT NULL;

ALTER TABLE "orgs"
	ALTER COLUMN "created_at" SET NOT NULL,
	ALTER COLUMN "updated_at" SET NOT NULL;

ALTER TABLE "user_tags"
	ALTER COLUMN "created_at" SET NOT NULL,
	ALTER COLUMN "updated_at" SET NOT NULL;

ALTER TABLE "users"
	ALTER COLUMN "created_at" SET NOT NULL,
	ALTER COLUMN "updated_at" SET NOT NULL;

ALTER TABLE "users" ALTER COLUMN deleted_at TYPE TIMESTAMP WITH TIME ZONE;

