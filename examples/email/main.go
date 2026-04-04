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

