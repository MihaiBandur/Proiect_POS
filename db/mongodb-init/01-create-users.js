const env = process.env;

const required = [
  "MONGO_DB_NAME",
  "MONGO_USER_ADMIN_USERNAME", "MONGO_USER_ADMIN_PASSWORD",
  "MONGO_DB_OWNER_USERNAME", "MONGO_DB_OWNER_PASSWORD",
  "MONGO_APP_USERNAME", "MONGO_APP_PASSWORD",
];
const missing = required.filter((name) => !env[name]);
if (missing.length > 0) {
  throw new Error("Lipsesc variabilele de mediu: " + missing.join(", "));
}


db.getSiblingDB("admin").createUser({
  user: env.MONGO_USER_ADMIN_USERNAME,
  pwd: env.MONGO_USER_ADMIN_PASSWORD,
  roles: [{ role: "userAdminAnyDatabase", db: "admin" }],
});

const projectDb = db.getSiblingDB(env.MONGO_DB_NAME);

projectDb.createUser({
  user: env.MONGO_DB_OWNER_USERNAME,
  pwd: env.MONGO_DB_OWNER_PASSWORD,
  roles: [{ role: "dbOwner", db: env.MONGO_DB_NAME }],
});

projectDb.createUser({
  user: env.MONGO_APP_USERNAME,
  pwd: env.MONGO_APP_PASSWORD,
  roles: [{ role: "readWrite", db: env.MONGO_DB_NAME }],
});
