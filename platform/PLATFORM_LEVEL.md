# ./platform

**Folder with platform-level logic**. This directory contains all the platform-level logic that will build up the actual project, like _setting up the database_ or _session store_.

- `./platform/cache` folder with Redis client and session store (Redis-backed or in-memory)
- `./platform/database` folder with database configuration (GORM: SQLite by default, PostgreSQL via `SQL_DSN`) and `AutoMigrate` schema setup
- `./platform/gateway` payment provider contract + registry; providers (`./platform/midtrans`, `./platform/nowpayments`, future `./platform/<name>`) are discovered via the registry and declare behavior through the optional capabilities in `platform/gateway/contracts.go`
