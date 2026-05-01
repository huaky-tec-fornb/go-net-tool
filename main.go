package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/huaky-tec-fornb/go-net-tool/internal/service"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	svc := service.NewNetService()

	app := application.New(application.Options{
		Name:        "GoNetTool",
		Description: "TCP/UDP 网络调试助手",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	svc.SetApp(app)

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "GoNetTool - 网络调试助手",
		Width:            1024,
		Height:           768,
		MinWidth:         800,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(30, 30, 35),
		URL:              "/",
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
