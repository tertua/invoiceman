# Build and Deployment

## Rules
- Makefile orchestrates all tasks (test, build, dev-be, dev-fe, webui.check)
- VERSION file: single source of truth, sync to webui/package.json via npm run sync:version
- Version bumps only in release commits, never in feat/fix commits
- Branching: dev active, main stable, master production
- Promotion: make promote (dev→main ff-only), make promote-prod (main→master + v tag)
- No merge commits on dev/main/master (git pull --ff-only or --rebase)
- File size check: npm --prefix webui run check:size (CI enforces baseline)
- Pre-push checks: check:map, check:size, check:fixtures, sync:version --check

## Examples

### Makefile Test Target
```makefile
# Makefile:13-18
test: clean critic security lint
	go test -v -timeout 30s -coverprofile=cover.out -cover ./...
	go tool cover -func=cover.out

build: test
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) main.go
```

### Dev Targets
```makefile
# Makefile:23-28
dev-be:
	go build -ldflags="$(LDFLAGS)" -o /tmp/opencode/tupay . && /tmp/opencode/tupay

dev-fe:
	npm --prefix webui run dev -- --host 0.0.0.0

webui.check:
	npm --prefix webui run lint
```

### File Size Baseline Format
```json
// scripts/file-size-baseline.json:1-6
{
 "app/controllers/admin_controller.go": 185,
 "app/controllers/ai_controller.go": 249,
 "app/controllers/ai_locale_test.go": 64,
 "app/controllers/audit_helper.go": 23,
 "app/controllers/auth_controller.go": 549,
```

## Anti-patterns

### ❌ Version Bump in Feature Commit
```bash
# BAD: bump version in feat commit
git commit -m "feat: add payment gateway"
# VERSION changed from 1.2.3 to 1.3.0

# GOOD: version bump in separate release commit
git commit -m "feat: add payment gateway"
# later, in release commit:
git commit -m "chore: release v1.3.0"
```

### ❌ Direct Commit to main/master
```bash
# BAD: commit directly to main
git checkout main
git commit -m "fix: urgent bug"

# GOOD: work on dev, promote via make promote
git checkout dev
git commit -m "fix: urgent bug"
make promote
```

### ❌ Growing File Past Baseline
```bash
# BAD: add 100 lines to invoice_controller.go (capped at 112/400)
# without checking if it passes check:size

# GOOD: run check before commit
npm --prefix webui run check:size
```
