# Email Example Overhaul — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the email example into a transactional email system for a fictional SaaS product ("Grove Cloud"), with 5 email templates, a preview server with multiple data scenarios, and thorough demonstration of hoist, capture, let/set scoping, and macro composition.

**Architecture:** Data loaded from JSON files (users, orders, scenarios). Templates use email-safe HTML (table-based layout, inline styles). Each email template extends a base layout and uses imported helper macros. Preview server allows viewing emails with different user/scenario combinations via query params.

**Tech Stack:** Go 1.24, Grove template engine, Chi router, JSON data files

**Spec:** `docs/superpowers/specs/2026-04-04-examples-expansion-design.md` — Example 4: Email

---

### Task 1: Create JSON data files

**Files:**
- Create: `examples/email/data/users.json`
- Create: `examples/email/data/orders.json`
- Create: `examples/email/data/scenarios.json`

- [ ] **Step 1: Create the data directory**

```bash
mkdir -p examples/email/data
```

- [ ] **Step 2: Write users.json**

```json
[
  {
    "id": 1,
    "name": "Alice Johnson",
    "email": "alice@example.com",
    "plan": "pro",
    "joined_date": "2025-11-15",
    "usage_pct": 72,
    "team_name": "Acme Corp"
  },
  {
    "id": 2,
    "name": "Bob Martinez",
    "email": "bob@example.com",
    "plan": "free",
    "joined_date": "2026-03-28",
    "usage_pct": 45,
    "team_name": ""
  },
  {
    "id": 3,
    "name": "Chen Wei",
    "email": "chen@example.com",
    "plan": "enterprise",
    "joined_date": "2025-06-01",
    "usage_pct": 91,
    "team_name": "GlobalTech Industries"
  }
]
```

- [ ] **Step 3: Write orders.json**

```json
[
  {
    "id": "ORD-2026-1847",
    "user_id": 1,
    "items": [
      {"name": "Pro Plan (Annual)", "quantity": 1, "price": 19900},
      {"name": "Additional Team Seat", "quantity": 3, "price": 4900}
    ],
    "total": 34600,
    "date": "2026-04-01"
  },
  {
    "id": "ORD-2026-1923",
    "user_id": 3,
    "items": [
      {"name": "Enterprise Plan (Annual)", "quantity": 1, "price": 99900},
      {"name": "Priority Support Add-on", "quantity": 1, "price": 29900},
      {"name": "Additional Storage (100GB)", "quantity": 2, "price": 9900}
    ],
    "total": 149600,
    "date": "2026-04-03"
  },
  {
    "id": "ORD-2026-2001",
    "user_id": 2,
    "items": [],
    "total": 0,
    "date": "2026-04-04"
  }
]
```

- [ ] **Step 4: Write scenarios.json**

Scenarios provide override data for previewing different states.

```json
{
  "default": {
    "reset_link": "https://app.grovecloud.io/reset?token=abc123def456",
    "reset_expiry": "24 hours",
    "old_plan": "free",
    "new_plan": "pro",
    "usage_limit": 10000,
    "usage_current": 7200
  },
  "expired_token": {
    "reset_link": "https://app.grovecloud.io/reset?token=expired789",
    "reset_expiry": "expired",
    "old_plan": "free",
    "new_plan": "pro",
    "usage_limit": 10000,
    "usage_current": 7200
  },
  "downgrade": {
    "reset_link": "https://app.grovecloud.io/reset?token=abc123def456",
    "reset_expiry": "24 hours",
    "old_plan": "enterprise",
    "new_plan": "pro",
    "usage_limit": 10000,
    "usage_current": 7200
  },
  "critical_usage": {
    "reset_link": "https://app.grovecloud.io/reset?token=abc123def456",
    "reset_expiry": "24 hours",
    "old_plan": "free",
    "new_plan": "pro",
    "usage_limit": 10000,
    "usage_current": 9500
  }
}
```

- [ ] **Step 5: Commit data files**

```bash
git add examples/email/data/
git commit -m "email: Add JSON data files for users, orders, and preview scenarios"
```

---

### Task 2: Rewrite main.go

**Files:**
- Modify: `examples/email/main.go`

- [ ] **Step 1: Write the complete main.go**

Replace the entire file. The new main.go includes:
- `User`, `OrderItem`, `Order` structs with JSON tags and GroveResolve
- JSON data loading from `data/` directory
- Scenario data loading and merging
- Custom `currency` filter
- Handlers: index (template list), preview (with user/scenario query params), source
- Each email template has a default data function that builds its template data

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	grove "grove/pkg/grove"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// --- Types ---

type User struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Plan       string `json:"plan"`
	JoinedDate string `json:"joined_date"`
	UsagePct   int    `json:"usage_pct"`
	TeamName   string `json:"team_name"`
}

func (u User) GroveResolve(key string) (any, bool) {
	switch key {
	case "id":
		return u.ID, true
	case "name":
		return u.Name, true
	case "email":
		return u.Email, true
	case "plan":
		return u.Plan, true
	case "joined_date":
		return u.JoinedDate, true
	case "usage_pct":
		return u.UsagePct, true
	case "team_name":
		return u.TeamName, true
	}
	return nil, false
}

type OrderItem struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    int    `json:"price"`
}

func (o OrderItem) GroveResolve(key string) (any, bool) {
	switch key {
	case "name":
		return o.Name, true
	case "quantity":
		return o.Quantity, true
	case "price":
		return o.Price, true
	case "line_total":
		return o.Price * o.Quantity, true
	}
	return nil, false
}

type Order struct {
	ID     string      `json:"id"`
	UserID int         `json:"user_id"`
	Items  []OrderItem `json:"items"`
	Total  int         `json:"total"`
	Date   string      `json:"date"`
}

func (o Order) GroveResolve(key string) (any, bool) {
	switch key {
	case "id":
		return o.ID, true
	case "items":
		items := make([]any, len(o.Items))
		for i, item := range o.Items {
			items[i] = item
		}
		return items, true
	case "total":
		return o.Total, true
	case "date":
		return o.Date, true
	}
	return nil, false
}

// --- Data ---

var (
	users     []User
	userMap   map[int]User
	orders    []Order
	orderMap  map[int]Order // keyed by user_id
	scenarios map[string]map[string]any
)

func loadJSON(baseDir, filename string, v any) {
	data, err := os.ReadFile(filepath.Join(baseDir, "data", filename))
	if err != nil {
		log.Fatalf("Failed to load %s: %v", filename, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		log.Fatalf("Failed to parse %s: %v", filename, err)
	}
}

func loadData(baseDir string) {
	loadJSON(baseDir, "users.json", &users)
	loadJSON(baseDir, "orders.json", &orders)

	// Load scenarios as raw JSON
	var rawScenarios map[string]map[string]any
	loadJSON(baseDir, "scenarios.json", &rawScenarios)
	scenarios = rawScenarios

	userMap = make(map[int]User)
	for _, u := range users {
		userMap[u.ID] = u
	}
	orderMap = make(map[int]Order)
	for _, o := range orders {
		orderMap[o.UserID] = o
	}
}

// --- Email template data builders ---

type emailTemplate struct {
	Name        string
	Label       string
	Description string
	BuildData   func(user User, scenario map[string]any) grove.Data
}

var emailTemplates = []emailTemplate{
	{
		Name:        "welcome",
		Label:       "Welcome Email",
		Description: "Sent when a new user creates an account.",
		BuildData: func(user User, scenario map[string]any) grove.Data {
			return grove.Data{"user": user}
		},
	},
	{
		Name:        "order-confirmation",
		Label:       "Order Confirmation",
		Description: "Sent after a successful purchase.",
		BuildData: func(user User, scenario map[string]any) grove.Data {
			order := orderMap[user.ID]
			return grove.Data{"user": user, "order": order}
		},
	},
	{
		Name:        "password-reset",
		Label:       "Password Reset",
		Description: "Sent when a user requests a password reset.",
		BuildData: func(user User, scenario map[string]any) grove.Data {
			return grove.Data{
				"user":         user,
				"reset_link":   scenario["reset_link"],
				"reset_expiry": scenario["reset_expiry"],
			}
		},
	},
	{
		Name:        "plan-change",
		Label:       "Plan Change Notification",
		Description: "Sent when a user upgrades or downgrades their plan.",
		BuildData: func(user User, scenario map[string]any) grove.Data {
			return grove.Data{
				"user":     user,
				"old_plan": scenario["old_plan"],
				"new_plan": scenario["new_plan"],
			}
		},
	},
	{
		Name:        "usage-alert",
		Label:       "Usage Alert",
		Description: "Sent when a user approaches their plan's usage limits.",
		BuildData: func(user User, scenario map[string]any) grove.Data {
			limit, _ := scenario["usage_limit"].(float64)
			current, _ := scenario["usage_current"].(float64)
			return grove.Data{
				"user":          user,
				"usage_limit":   int(limit),
				"usage_current": int(current),
				"usage_pct":     int((current / limit) * 100),
			}
		},
	},
}

var emailTemplateMap map[string]emailTemplate

func init() {
	emailTemplateMap = make(map[string]emailTemplate)
	for _, et := range emailTemplates {
		emailTemplateMap[et.Name] = et
	}
}

// --- Handlers ---

func indexHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		links := make([]any, len(emailTemplates))
		for i, et := range emailTemplates {
			links[i] = map[string]any{
				"name":        et.Name,
				"label":       et.Label,
				"description": et.Description,
			}
		}
		// Build user options for the preview form
		userOpts := make([]any, len(users))
		for i, u := range users {
			userOpts[i] = map[string]any{
				"id":   u.ID,
				"name": u.Name,
				"plan": u.Plan,
			}
		}
		result, err := eng.Render(r.Context(), "index.grov", grove.Data{
			"templates": links,
			"users":     userOpts,
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, result.Body)
	}
}

func previewHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		et, ok := emailTemplateMap[name]
		if !ok {
			http.NotFound(w, r)
			return
		}

		// Resolve user (default to first)
		user := users[0]
		if uidStr := r.URL.Query().Get("user"); uidStr != "" {
			if uid, err := strconv.Atoi(uidStr); err == nil {
				if u, ok := userMap[uid]; ok {
					user = u
				}
			}
		}

		// Resolve scenario (default to "default")
		scenarioName := r.URL.Query().Get("scenario")
		if scenarioName == "" {
			scenarioName = "default"
		}
		scenario := scenarios["default"]
		if s, ok := scenarios[scenarioName]; ok {
			// Merge: start with default, overlay specific scenario
			merged := make(map[string]any)
			for k, v := range scenarios["default"] {
				merged[k] = v
			}
			for k, v := range s {
				merged[k] = v
			}
			scenario = merged
		}

		data := et.BuildData(user, scenario)
		data["current_year"] = "2026"

		result, err := eng.Render(r.Context(), name+".grov", data)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, result.Body)
	}
}

func sourceHandler(eng *grove.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		templateDir := filepath.Join(filepath.Dir(mustThisFile()), "templates")
		content, err := os.ReadFile(filepath.Join(templateDir, name+".grov"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(content)
	}
}

func mustThisFile() string {
	_, f, _, _ := runtime.Caller(0)
	return f
}

// --- Main ---

func main() {
	_, thisFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(thisFile)

	loadData(baseDir)

	templateDir := filepath.Join(baseDir, "templates")
	store := grove.NewFileSystemStore(templateDir)
	eng := grove.New(grove.WithStore(store))
	eng.SetGlobal("current_year", "2026")

	eng.RegisterFilter("currency", grove.FilterFn(func(v grove.Value, args []grove.Value) (grove.Value, error) {
		cents, _ := v.ToInt64()
		dollars := cents / 100
		remainder := int(math.Abs(float64(cents % 100)))
		return grove.StringValue(fmt.Sprintf("$%d.%02d", dollars, remainder)), nil
	}))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", indexHandler(eng))
	r.Get("/preview/{name}", previewHandler(eng))
	r.Get("/source/{name}", sourceHandler(eng))

	fmt.Println("Grove Email listening on http://localhost:3003")
	log.Fatal(http.ListenAndServe(":3003", r))
}

var (
	_ interface{ GroveResolve(string) (any, bool) } = User{}
	_ interface{ GroveResolve(string) (any, bool) } = OrderItem{}
	_ interface{ GroveResolve(string) (any, bool) } = Order{}
)
```

- [ ] **Step 2: Verify it compiles**

```bash
cd examples/email && go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add examples/email/main.go
git commit -m "email: Rewrite main.go with data loading, scenarios, and preview handlers"
```

---

### Task 3: Create email templates

**Files:**
- Modify: `examples/email/templates/base-email.grov`
- Modify: `examples/email/templates/helpers.grov`
- Modify: `examples/email/templates/index.grov`
- Modify: `examples/email/templates/welcome.grov`
- Modify: `examples/email/templates/order-confirmation.grov`
- Modify: `examples/email/templates/password-reset.grov`
- Create: `examples/email/templates/plan-change.grov`
- Create: `examples/email/templates/usage-alert.grov`

- [ ] **Step 1: Rewrite base-email.grov**

Email-safe HTML layout with table-based structure and inline styles.

```
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{% block title %}Grove Cloud{% endblock %}</title>
  <style>
    body { margin: 0; padding: 0; background-color: #f4f4f7; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; }
    .wrapper { width: 100%; background-color: #f4f4f7; padding: 24px 0; }
    .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; }
    .header { background-color: #2E6740; padding: 24px; text-align: center; }
    .header h1 { color: #ffffff; margin: 0; font-size: 24px; }
    .body { padding: 32px 24px; color: #333333; line-height: 1.6; }
    .body h2 { color: #2E6740; margin: 0 0 16px; }
    .body p { margin: 0 0 16px; }
    .footer { padding: 24px; text-align: center; color: #888888; font-size: 12px; background-color: #f9f9f9; }
    .footer a { color: #2E6740; }
    .preheader { display: none !important; max-height: 0; overflow: hidden; mso-hide: all; }
  </style>
</head>
<body>
  <div class="preheader">{% block preheader %}{% endblock %}</div>
  <div class="wrapper">
    <div class="container">
      <div class="header">
        <h1>Grove Cloud</h1>
      </div>
      <div class="body">
        {% block body %}{% endblock %}
      </div>
      <div class="footer">
        {% block footer %}
          <p>&copy; {{ current_year }} Grove Cloud. All rights reserved.</p>
          <p><a href="https://app.grovecloud.io/settings">Manage preferences</a> &middot; <a href="https://app.grovecloud.io/help">Help Center</a></p>
        {% endblock %}
      </div>
    </div>
  </div>
</body>
</html>
```

- [ ] **Step 2: Rewrite helpers.grov**

Expanded macro library demonstrating macro composition.

```
{% macro button(text, href, color) %}
  {% if not color %}{% set color = "#2E6740" %}{% endif %}
  <table cellpadding="0" cellspacing="0" border="0" style="margin: 16px 0;">
    <tr>
      <td style="background-color: {{ color }}; border-radius: 6px; padding: 12px 24px;">
        <a href="{{ href }}" style="color: #ffffff; text-decoration: none; font-weight: bold; display: inline-block;">{{ text }}</a>
      </td>
    </tr>
  </table>
{% endmacro %}

{% macro divider() %}
  <hr style="border: none; border-top: 1px solid #e0e0e0; margin: 24px 0;">
{% endmacro %}

{% macro spacer(height) %}
  {% if not height %}{% set height = 16 %}{% endif %}
  <div style="height: {{ height }}px;"></div>
{% endmacro %}

{% macro heading(text) %}
  <h2 style="color: #2E6740; margin: 0 0 16px; font-size: 20px;">{{ text }}</h2>
{% endmacro %}

{% macro usage_bar(pct, color) %}
  {% if not color %}
    {% if pct >= 90 %}
      {% set color = "#dc3545" %}
    {% elif pct >= 75 %}
      {% set color = "#ffc107" %}
    {% else %}
      {% set color = "#2E6740" %}
    {% endif %}
  {% endif %}
  <div style="background-color: #e9ecef; border-radius: 4px; height: 24px; margin: 8px 0; overflow: hidden;">
    <div style="background-color: {{ color }}; height: 100%; width: {{ pct }}%; border-radius: 4px; transition: width 0.3s;"></div>
  </div>
{% endmacro %}

{% macro info_row(label, value) %}
  <tr>
    <td style="padding: 8px 0; color: #666666; width: 140px;">{{ label }}</td>
    <td style="padding: 8px 0; font-weight: bold;">{{ value }}</td>
  </tr>
{% endmacro %}
```

- [ ] **Step 3: Rewrite index.grov**

Preview server landing page with template list and user/scenario selectors.

```
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Grove Cloud Email Templates</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; color: #333; }
    h1 { color: #2E6740; }
    .template-list { list-style: none; padding: 0; }
    .template-item { border: 1px solid #e0e0e0; border-radius: 8px; padding: 16px; margin: 12px 0; }
    .template-item h3 { margin: 0 0 4px; }
    .template-item p { margin: 0 0 8px; color: #666; }
    .template-links { display: flex; gap: 12px; flex-wrap: wrap; }
    .template-links a { color: #2E6740; font-weight: bold; text-decoration: none; }
    .template-links a:hover { text-decoration: underline; }
  </style>
</head>
<body>
  <h1>Grove Cloud Email Templates</h1>
  <p>Preview transactional email templates with different users and scenarios.</p>

  <ul class="template-list">
    {% for tpl in templates %}
      <li class="template-item">
        <h3>{{ tpl.label }}</h3>
        <p>{{ tpl.description }}</p>
        <div class="template-links">
          {% for user in users %}
            <a href="/preview/{{ tpl.name }}?user={{ user.id }}">{{ user.name }} ({{ user.plan }})</a>
          {% endfor %}
          <a href="/source/{{ tpl.name }}">View Source</a>
        </div>
      </li>
    {% endfor %}
  </ul>
</body>
</html>
```

- [ ] **Step 4: Rewrite welcome.grov**

Demonstrates `hoist` for preheader and `capture` for reusable greeting block.

```
{% extends "base-email.grov" %}
{% import "helpers.grov" as h %}

{% block title %}Welcome to Grove Cloud{% endblock %}

{% hoist target="preheader" %}Welcome aboard, {{ user.name | default("there") }}! Here's how to get started with Grove Cloud.{% endhoist %}

{% block body %}
  {% capture greeting_block %}
    <p style="font-size: 18px;">Hi {{ user.name | default("there") }},</p>
    <p>Welcome to <strong>Grove Cloud</strong>! We're excited to have you on board.</p>
  {% endcapture %}

  {{ greeting_block | safe }}

  {{ h.divider() }}

  {{ h.heading("Get Started in 3 Steps") }}

  <table cellpadding="0" cellspacing="0" border="0" width="100%">
    {% let %}
      steps = [
        {"num": "1", "title": "Set up your workspace", "desc": "Create your first project and invite your team."},
        {"num": "2", "title": "Connect your tools", "desc": "Integrate with GitHub, Slack, and your CI pipeline."},
        {"num": "3", "title": "Deploy your first template", "desc": "Use our CLI or API to push templates to production."}
      ]
    {% endlet %}
    {% for step in steps %}
      <tr>
        <td style="padding: 12px 0; vertical-align: top;">
          <span style="display: inline-block; width: 32px; height: 32px; background-color: #2E6740; color: #fff; border-radius: 50%; text-align: center; line-height: 32px; font-weight: bold; margin-right: 12px;">{{ step.num }}</span>
        </td>
        <td style="padding: 12px 0;">
          <strong>{{ step.title }}</strong><br>
          <span style="color: #666;">{{ step.desc }}</span>
        </td>
      </tr>
    {% endfor %}
  </table>

  {{ h.spacer(8) }}
  {{ h.button("Go to Dashboard", "https://app.grovecloud.io/dashboard") }}

  {{ h.divider() }}
  <p style="color: #666; font-size: 14px;">Questions? Reply to this email or visit our <a href="https://app.grovecloud.io/help" style="color: #2E6740;">Help Center</a>.</p>
{% endblock %}
```

- [ ] **Step 5: Rewrite order-confirmation.grov**

Demonstrates for/empty loops with line item totals and arithmetic.

```
{% extends "base-email.grov" %}
{% import "helpers.grov" as h %}

{% block title %}Order Confirmation — {{ order.id }}{% endblock %}

{% hoist target="preheader" %}Your order {{ order.id | upper }} has been confirmed. Total: {{ order.total | currency }}.{% endhoist %}

{% block body %}
  <p>Hi {{ user.name | default("Customer") }},</p>
  <p>Thank you for your order! Here's your receipt:</p>

  {{ h.divider() }}

  {{ h.heading("Order " ~ order.id | upper) }}

  <table cellpadding="0" cellspacing="0" border="0" width="100%" style="border-collapse: collapse;">
    <thead>
      <tr style="border-bottom: 2px solid #2E6740;">
        <th style="text-align: left; padding: 8px 0;">Item</th>
        <th style="text-align: center; padding: 8px 0;">Qty</th>
        <th style="text-align: right; padding: 8px 0;">Price</th>
        <th style="text-align: right; padding: 8px 0;">Total</th>
      </tr>
    </thead>
    <tbody>
      {% for item in order.items %}
        <tr style="border-bottom: 1px solid #e0e0e0;">
          <td style="padding: 12px 0;">{{ item.name }}</td>
          <td style="padding: 12px 0; text-align: center;">{{ item.quantity }}</td>
          <td style="padding: 12px 0; text-align: right;">{{ item.price | currency }}</td>
          <td style="padding: 12px 0; text-align: right; font-weight: bold;">{{ item.line_total | currency }}</td>
        </tr>
      {% empty %}
        <tr>
          <td colspan="4" style="padding: 12px 0; text-align: center; color: #666;">No items in this order.</td>
        </tr>
      {% endfor %}
    </tbody>
  </table>

  {{ h.spacer(8) }}

  <table cellpadding="0" cellspacing="0" border="0" width="100%">
    <tr>
      <td style="text-align: right; padding: 8px 0; font-size: 18px; font-weight: bold;">
        Total: {{ order.total | currency }}
      </td>
    </tr>
  </table>

  {{ h.divider() }}

  {{ h.button("View Order", "https://app.grovecloud.io/orders/" ~ order.id) }}

  <p style="color: #666; font-size: 14px;">A receipt has been sent to {{ user.email }}.</p>
{% endblock %}
```

- [ ] **Step 6: Rewrite password-reset.grov**

Demonstrates capture for building conditional greeting and safe filter.

```
{% extends "base-email.grov" %}
{% import "helpers.grov" as h %}

{% block title %}Reset Your Password{% endblock %}

{% hoist target="preheader" %}Password reset requested for your Grove Cloud account.{% endhoist %}

{% block body %}
  {% capture greeting %}
    {% if user.name %}
      <p>Hi {{ user.name }},</p>
    {% else %}
      <p>Hi there,</p>
    {% endif %}
  {% endcapture %}

  {{ greeting | safe }}

  <p>We received a request to reset the password for your Grove Cloud account ({{ user.email }}).</p>

  {% if reset_expiry == "expired" %}
    <div style="background-color: #fff3cd; border: 1px solid #ffc107; border-radius: 6px; padding: 16px; margin: 16px 0;">
      <strong>This link has expired.</strong> Please request a new password reset from the login page.
    </div>
  {% else %}
    {{ h.button("Reset Password", reset_link) }}
    <p style="color: #666; font-size: 14px;">This link expires in {{ reset_expiry }}. If you didn't request this, you can safely ignore this email.</p>
  {% endif %}

  {{ h.divider() }}

  <p style="color: #666; font-size: 14px;"><strong>Security tip:</strong> Grove Cloud will never ask for your password via email. If you didn't request this reset, your account is still secure.</p>
{% endblock %}

{% block footer %}
  <p>&copy; {{ current_year }} Grove Cloud. This is an automated security notification.</p>
{% endblock %}
```

- [ ] **Step 7: Write plan-change.grov**

Demonstrates let blocks for computing comparison data.

```
{% extends "base-email.grov" %}
{% import "helpers.grov" as h %}

{% block title %}Plan Change Confirmation{% endblock %}

{% hoist target="preheader" %}Your Grove Cloud plan has been changed from {{ old_plan | title }} to {{ new_plan | title }}.{% endhoist %}

{% block body %}
  <p>Hi {{ user.name | default("there") }},</p>

  {% let %}
    is_upgrade = (new_plan == "pro" and old_plan == "free") or (new_plan == "enterprise" and (old_plan == "free" or old_plan == "pro"))
  {% endlet %}

  {% if is_upgrade %}
    <p>Great news! Your account has been <strong>upgraded</strong>.</p>
  {% else %}
    <p>Your plan has been <strong>changed</strong>. The new plan takes effect at the end of your current billing period.</p>
  {% endif %}

  {{ h.divider() }}
  {{ h.heading("Plan Comparison") }}

  <table cellpadding="0" cellspacing="0" border="0" width="100%" style="border-collapse: collapse;">
    {{ h.info_row("Previous Plan", old_plan | title) }}
    {{ h.info_row("New Plan", new_plan | title) }}
    {% if user.team_name %}
      {{ h.info_row("Team", user.team_name) }}
    {% endif %}
  </table>

  {{ h.spacer(16) }}

  {% if is_upgrade %}
    {{ h.button("Explore New Features", "https://app.grovecloud.io/dashboard") }}
  {% else %}
    {{ h.button("View Plan Details", "https://app.grovecloud.io/settings/billing", "#666666") }}
  {% endif %}

  {{ h.divider() }}
  <p style="color: #666; font-size: 14px;">Questions about your plan? <a href="https://app.grovecloud.io/help" style="color: #2E6740;">Contact support</a>.</p>
{% endblock %}
```

- [ ] **Step 8: Write usage-alert.grov**

Demonstrates the usage_bar macro (macro calling macro pattern) and conditional urgency.

```
{% extends "base-email.grov" %}
{% import "helpers.grov" as h %}

{% block title %}Usage Alert — Grove Cloud{% endblock %}

{% hoist target="preheader" %}You've used {{ usage_pct }}% of your {{ user.plan | title }} plan limits.{% endhoist %}

{% block body %}
  <p>Hi {{ user.name | default("there") }},</p>

  {% if usage_pct >= 90 %}
    {% set urgency = "critical" %}
    <div style="background-color: #f8d7da; border: 1px solid #dc3545; border-radius: 6px; padding: 16px; margin: 0 0 16px;">
      <strong style="color: #dc3545;">Critical:</strong> You're approaching your plan limit. Services may be restricted once you reach 100%.
    </div>
  {% elif usage_pct >= 75 %}
    {% set urgency = "warning" %}
    <div style="background-color: #fff3cd; border: 1px solid #ffc107; border-radius: 6px; padding: 16px; margin: 0 0 16px;">
      <strong style="color: #856404;">Heads up:</strong> You're using a significant portion of your plan allocation.
    </div>
  {% else %}
    {% set urgency = "info" %}
    <p>Here's a summary of your current usage:</p>
  {% endif %}

  {{ h.divider() }}
  {{ h.heading("Usage Summary") }}

  <table cellpadding="0" cellspacing="0" border="0" width="100%" style="border-collapse: collapse;">
    {{ h.info_row("Plan", user.plan | title) }}
    {{ h.info_row("Current Usage", usage_current ~ " / " ~ usage_limit ~ " requests") }}
    {{ h.info_row("Usage", usage_pct ~ "%") }}
  </table>

  {{ h.spacer(8) }}
  {{ h.usage_bar(usage_pct) }}
  {{ h.spacer(16) }}

  {% capture cta_block %}
    {% if urgency == "critical" %}
      {{ h.button("Upgrade Now", "https://app.grovecloud.io/settings/billing", "#dc3545") }}
    {% elif urgency == "warning" %}
      {{ h.button("Review Usage", "https://app.grovecloud.io/usage") }}
      {{ h.spacer(8) }}
      {{ h.button("Upgrade Plan", "https://app.grovecloud.io/settings/billing", "#666666") }}
    {% else %}
      {{ h.button("View Details", "https://app.grovecloud.io/usage") }}
    {% endif %}
  {% endcapture %}

  {{ cta_block | safe }}

  {{ h.divider() }}
  <p style="color: #666; font-size: 14px;">You're receiving this because usage alerts are enabled. <a href="https://app.grovecloud.io/settings/notifications" style="color: #2E6740;">Manage alert preferences</a>.</p>
{% endblock %}
```

- [ ] **Step 9: Commit**

```bash
git add examples/email/templates/
git commit -m "email: Add all email templates — welcome, order, password reset, plan change, usage alert"
```

---

### Task 4: Build and verify

- [ ] **Step 1: Build**

```bash
cd examples/email && go build ./...
```

- [ ] **Step 2: Run and verify routes**

```bash
cd examples/email && go run main.go &
sleep 2
curl -s http://localhost:3003/ | head -30
curl -s http://localhost:3003/preview/welcome | head -30
curl -s http://localhost:3003/preview/welcome?user=2 | head -30
curl -s http://localhost:3003/preview/order-confirmation?user=1 | head -30
curl -s http://localhost:3003/preview/order-confirmation?user=2 | head -30
curl -s http://localhost:3003/preview/password-reset?user=1&scenario=expired_token | head -30
curl -s http://localhost:3003/preview/plan-change?user=1&scenario=downgrade | head -30
curl -s http://localhost:3003/preview/usage-alert?user=3&scenario=critical_usage | head -30
curl -s http://localhost:3003/source/welcome | head -10
kill %1
```

Expected: All routes return HTML. Each email should show different content based on user and scenario. The welcome email for user 2 (Bob, free plan) should differ from user 1 (Alice, pro plan).

- [ ] **Step 3: Verify Grove features in rendered output**

Check that:
- Preheader text appears in the hidden div (from `hoist`)
- Captured greeting block renders correctly (from `capture` + `safe`)
- Plan change shows upgrade/downgrade messaging (from `let` + conditionals)
- Usage alert shows correct urgency level and colored bar (from macro composition)
- Order confirmation handles empty items (from `for`/`empty`)

- [ ] **Step 4: Final commit if any fixes needed**

```bash
git add examples/email/
git commit -m "email: Fix any issues found during verification"
```
