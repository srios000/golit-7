package main

import (
	"fmt"
	"log"
	"net/http"
	"pertemuan7_npm_release/configs"
	"pertemuan7_npm_release/handlers"
	"pertemuan7_npm_release/middlewares"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
    PORT := 8081

    configs.ConnectDB()
    if configs.DB == nil {
        log.Fatal("Database connection failed")
    }
    defer func() {
        if err := configs.DB.Close(); err != nil {
            log.Printf("Error closing database connection: %v", err)
        }
    }()

    mux := http.NewServeMux()
    fileServer := http.FileServer(http.Dir("catalog"))
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        handlers.ServeStaticFile(w, r, "catalog", fileServer)
    })
    mux.HandleFunc("/api/games/", handlers.HandleGames)
    mux.HandleFunc("/api/games", handlers.HandleGames)

    // Apply middleware
    loggedMux := middlewares.LogRequestHandler(mux)

    fmt.Printf("Server berjalan di http://localhost:%d\n", PORT)
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), loggedMux))
}