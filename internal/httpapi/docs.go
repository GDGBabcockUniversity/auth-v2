package httpapi

import (
	_ "embed"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

//go:embed openapi.yaml
var openapiYAML []byte

//go:embed openapi.json
var openapiJSON []byte

func registerDocs(app *fiber.App) {
	app.Get("/openapi.yaml", func(c fiber.Ctx) error {
		c.Set("Content-Type", "application/yaml; charset=utf-8")
		return c.Send(openapiYAML)
	})

	app.Get("/openapi.json", func(c fiber.Ctx) error {
		c.Set("Content-Type", "application/json; charset=utf-8")
		return c.Send(openapiJSON)
	})

	app.Get("/docs", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(swaggerUIHTML)
	})
	app.Get("/docs/", func(c fiber.Ctx) error {
		return c.Redirect().Status(http.StatusMovedPermanently).To("/docs")
	})
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>GDG Babcock Auth API — Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css">
  <style>
    body { margin: 0; background: #fafafa; }
    .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js" crossorigin></script>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-standalone-preset.js" crossorigin></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({
        url: "/openapi.json",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
        layout: "StandaloneLayout",
        tryItOutEnabled: true,
        persistAuthorization: true
      });
    };
  </script>
</body>
</html>
`