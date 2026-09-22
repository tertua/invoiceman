package utils

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
)

// DefaultPageSize is returned when ?per_page= is absent or invalid.
const DefaultPageSize = 20

// MaxPageSize caps ?per_page= to bound response sizes.
const MaxPageSize = 100

// Pagination holds a validated page request from ?page=&per_page=.
type Pagination struct {
	Page    int
	PerPage int
}

// ParsePagination reads ?page= (default 1) and ?per_page= (default 20,
// capped at 100). Invalid values fall back to defaults, never errors,
// so listing endpoints stay backward compatible.
func ParsePagination(c fiber.Ctx) Pagination {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage, err := strconv.Atoi(c.Query("per_page"))
	if err != nil || perPage < 1 {
		perPage = DefaultPageSize
	}
	if perPage > MaxPageSize {
		perPage = MaxPageSize
	}
	return Pagination{Page: page, PerPage: perPage}
}

// Limit returns the SQL LIMIT for this page.
func (p Pagination) Limit() int { return p.PerPage }

// Offset returns the SQL OFFSET for this page.
func (p Pagination) Offset() int { return (p.Page - 1) * p.PerPage }

// Meta builds the {"page","per_page","total","total_pages"} envelope.
// Data keys are unchanged; clients adopt pagination gradually.
func (p Pagination) Meta(total int64) fiber.Map {
	pages := total / int64(p.PerPage)
	if total%int64(p.PerPage) != 0 {
		pages++
	}
	return fiber.Map{
		"page":        p.Page,
		"per_page":    p.PerPage,
		"total":       total,
		"total_pages": pages,
	}
}
