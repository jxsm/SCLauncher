package main

import (
	"embed"

	"SCLauncher/backend/appinfo"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// wailsConfig 嵌入 wails.json，使版本号与 Wails 构建系统（exe/安装包版本
// 信息）共用唯一来源 info.productVersion
//
//go:embed wails.json
var wailsConfig []byte

func main() {
	// 从 wails.json 注入版本号（唯一来源，改版本号只改 wails.json）
	appinfo.SetWailsConfig(wailsConfig)

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "SCLauncher",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   "SCLauncher",
				Message: "© 2026 SurvivalCraft Launcher",
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:   false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
