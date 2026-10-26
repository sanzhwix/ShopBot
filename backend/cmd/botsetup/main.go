// botsetup fills in the bot's Telegram profile: name, descriptions and command
// lists (English and Russian) and, with -photo, the profile picture.
//
//	BOT_TOKEN=... go run ./cmd/botsetup -photo ../assets/bot/avatar.jpg
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	tg "github.com/go-telegram/bot"

	"shopbot/internal/bot"
)

func main() {
	photo := flag.String("photo", "", "JPEG file for the bot's profile picture (optional)")
	flag.Parse()

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN is empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	api, err := tg.New(token)
	if err != nil {
		log.Fatalf("telegram: %v", err)
	}
	me, err := api.GetMe(ctx)
	if err != nil {
		log.Fatalf("telegram: %v", err)
	}
	log.Printf("bot: @%s", me.Username)

	if err := bot.ApplyProfile(ctx, api); err != nil {
		log.Fatalf("profile: %v", err)
	}
	log.Print("name, descriptions and commands are set (English and Russian)")

	if *photo != "" {
		jpeg, err := os.ReadFile(*photo)
		if err != nil {
			log.Fatalf("photo: %v", err)
		}
		if err := bot.ApplyPhoto(ctx, api, jpeg); err != nil {
			log.Fatalf("photo: %v", err)
		}
		log.Printf("profile photo is set from %s", *photo)
	}
}
