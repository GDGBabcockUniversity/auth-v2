package httpapi

import (
	"embed"
	"io/fs"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

//go:embed web
var dashboardFiles embed.FS

// FirebaseWebConfig carries the Firebase Web SDK values the browser needs to run
// Google Sign-In. These are public client identifiers, not credentials; the
// service account that verifies the resulting ID tokens stays server-side.
type FirebaseWebConfig struct {
	APIKey     string `json:"apiKey"`
	AuthDomain string `json:"authDomain"`
	ProjectID  string `json:"projectId"`
}

// Configured reports whether the dashboard has enough to start a sign-in.
func (f FirebaseWebConfig) Configured() bool {
	return f.APIKey != "" && f.AuthDomain != "" && f.ProjectID != ""
}

// registerDashboard serves the member and admin dashboard from the embedded
// web directory. It shares an origin with the API, so dashboard requests never
// depend on CORS_ORIGINS.
func (a API) registerDashboard(app *fiber.App) {
	assets, err := fs.Sub(dashboardFiles, "web")
	if err != nil {
		panic(err)
	}

	app.Get("/dashboard/config.json", func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		return c.JSON(fiber.Map{
			"firebase":   a.Firebase,
			"configured": a.Firebase.Configured(),
		})
	})

	app.Get("/dashboard*", static.New("", static.Config{
		FS:         assets,
		IndexNames: []string{"index.html"},
	}))
}
