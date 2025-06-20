package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"hkd.nam2507/service"
	"log"
)

func main() {
	dg, err := discordgo.New("Bot " + service.Token)
	if err != nil {
		log.Fatalf("Lỗi tạo bot Discord: %v", err)
	}
	dg.AddHandler(service.MessageCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentMessageContent

	err = dg.Open()
	if err != nil {
		log.Fatalf("Không thể kết nối Discord: %v", err)
	}

	fmt.Println("Bot Discord đã sẵn sàng.")
	select {}
}
