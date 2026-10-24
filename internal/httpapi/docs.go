package httpapi

import (
	"github.com/gofiber/fiber/v3"
	"github.com/swaggo/swag"
)

func registerSwagger(app *fiber.App) {
	app.Get("/swagger/doc.json", func(c fiber.Ctx) error {
		doc, err := swag.ReadDoc()
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		c.Set("Content-Type", "application/json")
		return c.SendString(doc)
	})

	app.Get("/swagger", func(c fiber.Ctx) error {
		return c.Redirect().To("/swagger/index.html")
	})

	app.Get("/swagger/index.html", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(swaggerIndexHTML)
	})
}

const swaggerIndexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>GDG Babcock Auth API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui.css">
  <style>body{margin:0}.topbar{display:none}</style>
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
<script src="https://unpkg.com/swagger-ui-dist@5.17.14/swagger-ui-standalone-preset.js"></script>
<script>
window.onload = () => {
  SwaggerUIBundle({
    url: "/swagger/doc.json",
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
</html>`