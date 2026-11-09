package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"hkd.nam2507/service"
)

func main() {
	// Khởi chạy Web Server cho tính năng Thay đổi HKD (port 8080)
	go func() {
		port := ":8080"
		if err := service.StartWebServer(port); err != nil {
			log.Printf("Lỗi Web Server: %v", err)
		}
	}()

	// Discord Bot cho tính năng Đăng ký thành lập HKD (giữ nguyên không đổi)
	if service.Token == "" {
		log.Println("⚠️ DISCORD_BOT_TOKEN chưa được cài đặt trong .env. Web Server vẫn hoạt động bình thường tại http://localhost:8080")
		select {}
	}

	dg, err := discordgo.New("Bot " + service.Token)
	if err != nil {
		log.Printf("Lỗi tạo bot Discord: %v", err)
		select {}
	}
	dg.AddHandler(service.MessageCreate)
	dg.AddHandler(service.InteractionCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentMessageContent

	err = dg.Open()
	if err != nil {
		log.Printf("Không thể kết nối Discord: %v", err)
		select {}
	}

	service.RegisterSlashCommands(dg)
	fmt.Println("Bot Discord đã sẵn sàng.")
	select {}
}
