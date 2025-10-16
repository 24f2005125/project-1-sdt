package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	AsciiArt()

	if err := godotenv.Load(); err != nil {
		log.Fatal("⚠️  No .env file found (using system environment)")
	}

	if err := InitOpenAI(); err != nil {
		log.Fatal("⚠️  OpenAI error: ", err)
	}

	if err := InitGit(); err != nil {
		log.Fatal("⚠️  Git error: ", err)
	}

	if err := StartServer(fmt.Sprintf(":%s", os.Getenv("PORT"))); err != nil {
		log.Fatal(err)
	}
}
