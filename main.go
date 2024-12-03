package main

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/canhlinh/svg2png"
	"github.com/patrickmn/go-cache"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

//go:embed index.html svg.tmpl

var content embed.FS

var chrome *svg2png.Chrome

func main() {
	chrome = svg2png.NewChrome().SetHeight(600).SetWith(1200).SetTimeout(10 * time.Second)
	ch := cache.New(24*time.Hour, 48*time.Hour)

	app := fiber.New()
	app.Use(compress.New())
	app.Use(cors.New())
	app.Use(logger.New())

	app.Get("/", func(c *fiber.Ctx) error {
		c.Type("html", "UTF8")
		body, _ := content.ReadFile("index.html")
		return c.Send(body)
	})

	app.Get("/opengraph", func(c *fiber.Ctx) error {
		vars := map[string]string{
			"siteTitle": ensureDecoded(c.Query("siteTitle", "")),
			"title":     ensureDecoded(c.Query("title", "")),
			"tags":      ensureDecoded(c.Query("tags", "")),
			"image":     ensureDecoded(c.Query("image", "")),
			"twitter":   stripAt(ensureDecoded(c.Query("twitter", ""))),
			"bluesky":   stripAt(ensureDecoded(c.Query("bluesky", ""))),
			"fediverse": stripAt(ensureDecoded(c.Query("fediverse", ""))),
			"github":    stripAt(ensureDecoded(c.Query("github", ""))),
			"website":   ensureDecoded(c.Query("website", "")),
			"bgColor":   ensureDecoded(c.Query("bgColor", c.Query("bgColour", "#fff"))),
			"fgColor":   ensureDecoded(c.Query("fgColor", c.Query("fgColour", "#2B414D"))),
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
			ch.Set(key, png, cache.DefaultExpiration)
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

	if err := chrome.Screenshoot(fmt.Sprintf("file://%s", file.Name()), imageFile.Name()); err != nil {
		return nil, err
	}
	defer os.Remove(imageFile.Name())

	return os.ReadFile(imageFile.Name())
}

// Some sites (LinkedIn) encode the already encoded URL so we need to double-decode to be sure
func ensureDecoded(str string) string {
	decoded, err := url.QueryUnescape(str)
	if err != nil {
		return str
	}
	return decoded
}

func stripAt(str string) string {
	return strings.TrimPrefix(str, "@")
}
