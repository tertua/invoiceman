# Import Conventions

## Rules
- Backend: stdlib → third-party → internal, separated by blank lines
- Frontend: @ alias for webui/src (configured in Vite)
- Axios imports restricted to webui/src/api/http.js (CI enforces)
- @react-pdf/renderer restricted to 2 PDF files (CI enforces)
- recharts restricted to 3 chart components (CI enforces)
- GORM imported as gorm.io/gorm, never gorm.io/driver/*
- Frontend hooks from @tanstack/react-query for server state
- i18n imports from @/lib/i18n (re-export of language files)

## Examples

### Backend Import Structure
```go
// app/controllers/ai_controller.go:1-13
package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/tertua/tupay/app/models"
	"github.com/tertua/tupay/pkg/logger"
	"github.com/tertua/tupay/pkg/utils"
```

### Frontend @ Alias
```js
// webui/src/hooks/useInvoices.js:1-2
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { invoicesApi } from "@/api/invoices";
```

### Restricted Import (Axios)
```js
// webui/src/api/http.js:1-3
import axios from "axios";
import { localizeApiError } from "@/lib/utils";
import { broadcastCaptchaReset, shouldResetCaptcha } from "./captchaReset";
```

## Anti-patterns

### ❌ Axios Outside http.js
```js
// BAD: direct axios import (CI fails)
import axios from "axios";
axios.get("/api/invoices");

// GOOD: use apiClient
import { apiClient } from "@/api/http";
apiClient.get("/invoices");
```

### ❌ Mixing Import Styles
```go
// BAD: unsorted imports
import (
	"github.com/tertua/tupay/app/models"
	"errors"
	"github.com/gofiber/fiber/v3"
	"strings"
)

// GOOD: grouped and sorted
import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/tertua/tupay/app/models"
)
```
