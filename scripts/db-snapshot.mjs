#!/usr/bin/env node
// Backup/restore for the app database: SQLite (SQL_DSN empty) or PostgreSQL (SQL_DSN=postgres://…).
// Usage: node scripts/db-snapshot.mjs backup | restore <file|latest>
//   restore: CONFIRM=yes skips the prompt; FORCE=yes ignores the PostgreSQL open-connection guard.
import { DatabaseSync } from "node:sqlite";
import { copyFileSync, existsSync, mkdirSync, readFileSync, readdirSync, rmSync, statSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { createInterface } from "node:readline";
import { spawnSync } from "node:child_process";

const root = resolve(fileURLToPath(new URL("..", import.meta.url)));
const backupDir = join(root, "data", "backups");
const PG_HINT = "install the PostgreSQL client tools first (apt install postgresql-client / brew install libpq)";

function fail(message) {
  console.error(`error: ${message}`);
  process.exit(1);
}

// env precedence mirrors godotenv: process.env wins over .env, which wins over the default.
function loadEnv() {
  const file = join(root, ".env");
  const values = {};
  if (existsSync(file)) {
    for (const line of readFileSync(file, "utf8").split(/\r?\n/)) {
      const match = line.match(/^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$/);
      if (match) values[match[1]] = match[2].trim().replace(/^(["'])(.*)\1$/, "$2");
    }
  }
  // An empty-but-set variable must win (mirrors godotenv), hence `in` not `||`.
  return (key, fallback = "") => (key in process.env ? process.env[key] : key in values ? values[key] : fallback);
}

// PG* env for the pg tools; the password travels via env, never argv (visible in ps).
function pgEnv(dsn) {
  let url;
  try {
    url = new URL(dsn);
  } catch {
    fail(`invalid SQL_DSN URL: ${dsn}`);
  }
  const database = decodeURIComponent(url.pathname.replace(/^\//, ""));
  if (!database) fail("SQL_DSN has no database name");
  const env = { PGHOST: url.hostname, PGPORT: url.port || "5432", PGDATABASE: database };
  if (url.username) env.PGUSER = decodeURIComponent(url.username);
  if (url.password) env.PGPASSWORD = decodeURIComponent(url.password);
  if (url.searchParams.get("sslmode")) env.PGSSLMODE = url.searchParams.get("sslmode");
  return env;
}

function config() {
  const env = loadEnv();
  const dsn = env("SQL_DSN").trim();
  if (!dsn) return { backend: "sqlite", path: resolve(root, env("SQLITE_PATH", "./data/db/tupay.db")) };
  if (dsn.startsWith("postgres://") || dsn.startsWith("postgresql://")) return { backend: "postgres", env: pgEnv(dsn) };
  fail(`unsupported SQL_DSN ${JSON.stringify(dsn)} — expected empty (SQLite) or a postgres:// URL`);
}

function tail(res) {
  return (res.stderr || res.stdout || "").trim().split("\n").slice(-5).join("\n");
}

function tryRun(tool, args, env = process.env) {
  const res = spawnSync(tool, args, { encoding: "utf8", env });
  if (res.error?.code === "ENOENT") fail(`${tool} not found — ${PG_HINT}`);
  return res;
}

function run(tool, args, env = process.env) {
  const res = tryRun(tool, args, env);
  if (res.status !== 0) fail(`${tool} failed: ${tail(res)}`);
  return res.stdout || "";
}

function snapshotName(prefix, ext) {
  const iso = new Date().toISOString().slice(0, 19).replace(/:/g, "").replace("T", "-");
  return `${prefix}-${iso}.${ext}`;
}

// VACUUM INTO writes a consistent, compacted copy while writers keep running.
function vacuumInto(sourcePath, targetPath) {
  if (existsSync(targetPath)) fail(`target already exists: ${targetPath}`);
  const db = new DatabaseSync(sourcePath, { readOnly: true });
  try {
    db.exec(`VACUUM INTO '${targetPath.replace(/'/g, "''")}'`);
  } finally {
    db.close();
  }
}

function integrityOk(path) {
  // A non-database file throws on open — treat it as corrupt, not as a crash.
  try {
    const db = new DatabaseSync(path, { readOnly: true });
    try {
      const row = db.prepare("PRAGMA integrity_check").get();
      return String(Object.values(row ?? {})[0] ?? "").toLowerCase() === "ok";
    } finally {
      db.close();
    }
  } catch {
    return false;
  }
}

function listBackups(ext) {
  if (!existsSync(backupDir)) return [];
  return readdirSync(backupDir)
    .filter((name) => name.endsWith(ext))
    .map((name) => join(backupDir, name))
    .sort((a, b) => statSync(b).mtimeMs - statSync(a).mtimeMs);
}

function inUse(path) {
  for (const tool of ["lsof", "fuser"]) {
    const probe = spawnSync(tool, [path], { encoding: "utf8" });
    if (probe.error) continue;
    if (tool === "lsof") return probe.status === 0 && probe.stdout.trim().length > 0;
    return probe.status === 0;
  }
  console.warn("warning: neither lsof nor fuser available — cannot verify the backend is stopped");
  return false;
}

async function confirm(question) {
  if (process.env.CONFIRM === "yes") return true;
  if (!process.stdin.isTTY) fail("refusing to restore without a TTY — rerun with CONFIRM=yes");
  const rl = createInterface({ input: process.stdin, output: process.stdout });
  const answer = await new Promise((done) => rl.question(`${question} `, done));
  rl.close();
  return answer.trim().toLowerCase() === "yes";
}

function reportSize(target) {
  return ` (${Math.round(statSync(target).size / 1024)} KiB)`;
}

function backupSqlite() {
  const source = config().path;
  if (!existsSync(source)) fail(`database not found: ${source} (start the backend once, or set SQLITE_PATH)`);
  mkdirSync(backupDir, { recursive: true });
  const target = join(backupDir, snapshotName("tupay", "db"));
  try {
    vacuumInto(source, target);
  } catch (error) {
    rmSync(target, { force: true });
    fail(`backup failed: ${error.message}`);
  }
  if (!integrityOk(target)) fail(`backup failed integrity check: ${target}`);
  console.log(`backup ok: ${target}${reportSize(target)}`);
}

function backupPostgres(env) {
  mkdirSync(backupDir, { recursive: true });
  const target = join(backupDir, snapshotName("tupay", "dump"));
  const dump = tryRun("pg_dump", ["--format=custom", "--no-password", `--file=${target}`], { ...process.env, ...env });
  if (dump.status !== 0) {
    rmSync(target, { force: true });
    fail(`pg_dump failed: ${tail(dump)}`);
  }
  // Custom-format archives list their TOC without touching the server — a corrupt file fails here.
  const probe = tryRun("pg_restore", ["--no-password", "--list", target]);
  if (probe.status !== 0) {
    rmSync(target, { force: true });
    fail(`pg_dump output is not a valid archive: ${target}`);
  }
  console.log(`backup ok: ${target}${reportSize(target)}`);
}

function restoreSqlite(fileArg) {
  const target = config().path;
  const source = pickBackup(fileArg, ".db", "sqlite");
  if (!integrityOk(source)) fail(`backup is corrupt, refusing to restore: ${source}`);
  if (existsSync(target) && inUse(target)) fail(`database is in use: ${target} — stop the backend first (Ctrl-C / make docker.stop)`);
  let safety = "";
  const proceed = async () => {
    if (existsSync(target)) {
      mkdirSync(backupDir, { recursive: true });
      safety = join(backupDir, snapshotName("pre-restore", "db"));
      vacuumInto(target, safety);
    }
    copyFileSync(source, target);
    // Stale WAL/journal sidecars from the replaced file would corrupt it on open.
    for (const suffix of ["-wal", "-shm", "-journal"]) rmSync(`${target}${suffix}`, { force: true });
    if (!integrityOk(target)) fail(`restored file failed integrity check — restore ${safety || "your backup"} back`);
  };
  return { source, target, proceed, done: () => {
    console.log(`restore ok: ${target}`);
    if (safety) console.log(`previous database saved as: ${safety}`);
    console.log("restart the backend to pick up the restored database");
  } };
}

function restorePostgres(fileArg, env) {
  const source = pickBackup(fileArg, ".dump", "postgres");
  const runEnv = { ...process.env, ...env };
  const probe = tryRun("pg_restore", ["--no-password", "--list", source], runEnv);
  if (probe.status !== 0) fail(`backup is unreadable, refusing to restore: ${source}`);
  if (process.env.FORCE !== "yes") {
    const sql = "SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND pid <> pg_backend_pid()";
    const open = Number(run("psql", ["--no-password", "--tuples-only", "--no-align", `--command=${sql}`], runEnv).trim());
    if (open > 0) fail(`database has ${open} other open connection(s) — stop the backend first (or rerun with FORCE=yes to ignore)`);
  }
  let safety = "";
  const proceed = async () => {
    mkdirSync(backupDir, { recursive: true });
    safety = join(backupDir, snapshotName("pre-restore", "dump"));
    const dump = tryRun("pg_dump", ["--format=custom", "--no-password", `--file=${safety}`], runEnv);
    if (dump.status !== 0) fail(`safety dump failed, refusing to restore: ${tail(dump)}`);
    // pg_restore ignores PGDATABASE — the target must come from --dbname.
    const restore = tryRun("pg_restore", ["--clean", "--if-exists", "--no-password", `--dbname=${env.PGDATABASE}`, source], runEnv);
    if (restore.status !== 0) fail(`pg_restore reported errors — inspect the database; previous state saved as:\n  ${safety}\n  ${tail(restore)}`);
    run("psql", ["--no-password", "--tuples-only", "--command=SELECT 1"], runEnv);
  };
  return { source, target: `postgres database ${env.PGDATABASE}@${env.PGHOST}:${env.PGPORT}`, proceed, done: () => {
    console.log(`restore ok: postgres database ${env.PGDATABASE}`);
    console.log(`previous database saved as: ${safety}`);
  } };
}

function pickBackup(fileArg, ext, backend) {
  if (!fileArg) fail(`usage: restore <file|latest>\navailable backups:\n  ${listBackups(ext).join("\n  ") || "(none)"}`);
  const source = fileArg === "latest" ? listBackups(ext)[0] : resolve(process.cwd(), fileArg);
  if (!source || !existsSync(source)) fail(`backup not found for the ${backend} backend: ${fileArg}`);
  return source;
}

async function restore(fileArg) {
  const cfg = config();
  const plan = cfg.backend === "sqlite" ? restoreSqlite(fileArg) : restorePostgres(fileArg, cfg.env);
  if (!(await confirm(`Replace ${plan.target} with ${plan.source}? Type "yes" to continue:`))) fail("aborted");
  await plan.proceed();
  plan.done();
}

const [command, fileArg] = process.argv.slice(2);
if (command === "backup") {
  const cfg = config();
  if (cfg.backend === "sqlite") backupSqlite();
  else backupPostgres(cfg.env);
} else if (command === "restore") await restore(fileArg);
else fail("usage: node scripts/db-snapshot.mjs backup | restore <file|latest>");
