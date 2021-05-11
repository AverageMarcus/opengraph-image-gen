package main

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/canhlinh/svg2png"
	"github.com/patrickmn/go-cache"

	"github.com/gofiber/fiber/v2"
	fiberCache "github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

//go:embed index.html svg.tmpl

var content embed.FS

func main() {
	app := fiber.New()
	app.Use(fiberCache.New())
	app.Use(compress.New())
	app.Use(cors.New())
	app.Use(logger.New())

	ch := cache.New(6*time.Hour, 24*time.Hour)

	app.Get("/", func(c *fiber.Ctx) error {
		c.Type("html", "UTF8")
		body, _ := content.ReadFile("index.html")
		return c.Send(body)
	})

	app.Get("/opengraph", func(c *fiber.Ctx) error {
		vars := map[string]string{
			"siteTitle": c.Query("siteTitle", ""),
			"title":     c.Query("title", ""),
			"tags":      c.Query("tags", ""),
			"image":     c.Query("image", ""),
			"twitter":   c.Query("twitter", ""),
			"github":    c.Query("github", ""),
			"website":   c.Query("website", ""),
			"bgColor":   c.Query("bgColor", c.Query("bgColour", "#fff")),
			"fgColor":   c.Query("fgColor", c.Query("fgColour", "#2B414D")),
		}

		key := generateKey(vars)

		png, found := ch.Get(key)
		if !found || len(png.([]byte)) == 0 {
			var err error
			png, err = generateImage(vars)
			if err != nil {
				fmt.Println(err)
				return c.SendStatus(500)
			}
			ch.Set(key, png, 0)
		}

		c.Type("png")
		return c.Send(png.([]byte))
	})

	app.Listen(":3000")
}

func generateKey(vars map[string]string) string {
	varsByte, _ := json.Marshal(vars)
	return base64.StdEncoding.EncodeToString(varsByte)
}

func generateImage(vars map[string]string) ([]byte, error) {
	file, err := os.CreateTemp(os.TempDir(), "img-*.html")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())

	t := template.Must(template.New("svg.tmpl").Funcs(template.FuncMap{
		"split": func(input string) []string {
			return strings.Split(input, ",")
		},
	}).ParseFS(content, "svg.tmpl"))
	t.Execute(file, vars)

	imageFile, err := os.CreateTemp(os.TempDir(), "img-*.png")

	chrome := svg2png.NewChrome().SetHeight(600).SetWith(1200)
	if err := chrome.Screenshoot(fmt.Sprintf("file://%s", file.Name()), imageFile.Name()); err != nil {
		return nil, err
	}
	defer os.Remove(imageFile.Name())

	return os.ReadFile(imageFile.Name())
}
