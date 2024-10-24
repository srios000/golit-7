package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

// Struct untuk game
type Game struct {
	ID        int     `json:"id_game"`
	Nama      string  `json:"nama_game"`
	Developer string  `json:"developer"`
	Genre     string  `json:"genre"`
	Harga     float64 `json:"harga"`
}

func connectDB() {
	var err error
	//! apabila user root ada password, tambahkan passsword di antara simbol titik dua dan simbol @ (contoh: root:password@tcp....)
	db, err = sql.Open("mysql", "root:@tcp(127.0.0.1:3308)/steam_catalog")
	if err != nil {
		log.Fatal(err)
	}
}

func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func handleGames(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleGames: %s %s", r.Method, r.URL.Path)
	path := r.URL.Path
	var id int
	var err error

	// Cek apakah URL cukup panjang untuk memiliki ID setelah "/api/games/"
	if len(path) > len("/api/games/") {
		id, err = strconv.Atoi(path[len("/api/games/"):])
		if err != nil {
			http.Error(w, "ID tidak valid", http.StatusBadRequest)
			return
		}
	} else {
		id = 0
	}

	switch r.Method {
	case http.MethodGet:
		if id == 0 {
			// Mendapatkan semua game
			rows, err := db.Query("SELECT id_game, nama_game, developer, genre, harga FROM games")
			if err != nil {
				http.Error(w, "Gagal mengambil data game", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var games []Game
			for rows.Next() {
				var game Game
				err := rows.Scan(&game.ID, &game.Nama, &game.Developer, &game.Genre, &game.Harga)
				if err != nil {
					http.Error(w, "Gagal memproses data game", http.StatusInternalServerError)
					return
				}
				games = append(games, game)
			}
			respondJSON(w, games)
		} else {
			var game Game
			err := db.QueryRow("SELECT id_game, nama_game, developer, genre, harga FROM games WHERE id_game = ?", id).
				Scan(&game.ID, &game.Nama, &game.Developer, &game.Genre, &game.Harga)
			if err != nil {
				http.Error(w, "Game tidak ditemukan", http.StatusNotFound)
				return
			}
			respondJSON(w, game)
		}

	case http.MethodPost:
		var game Game
		if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
			http.Error(w, "Data tidak valid", http.StatusBadRequest)
			return
		}
		if _, err := db.Exec("INSERT INTO games (nama_game, developer, genre, harga) VALUES (?, ?, ?, ?)", 
			game.Nama, game.Developer, game.Genre, game.Harga); err != nil {
			http.Error(w, "Gagal menambahkan game", http.StatusInternalServerError)
			return
		}
		respondJSON(w, map[string]string{"message": "Game berhasil ditambahkan!"})

	case http.MethodPut:
		var game Game
		if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
			http.Error(w, "Data tidak valid", http.StatusBadRequest)
			return
		}
		result, err := db.Exec("UPDATE games SET nama_game=?, developer=?, genre=?, harga=? WHERE id_game=?", 
			game.Nama, game.Developer, game.Genre, game.Harga, id)
		if err != nil {
			http.Error(w, "Gagal memperbarui game", http.StatusInternalServerError)
			return
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			http.Error(w, "Game dengan ID tersebut tidak ditemukan", http.StatusNotFound)
			return
		}
		respondJSON(w, map[string]string{"message": "Game berhasil diperbarui!"})	
	case http.MethodDelete:
		result, err := db.Exec("DELETE FROM games WHERE id_game=?", id)
		if err != nil {
			http.Error(w, "Gagal menghapus game", http.StatusInternalServerError)
			return
		}
	
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			http.Error(w, "Game dengan ID tersebut tidak ditemukan", http.StatusNotFound)
			return
		}
	
		respondJSON(w, map[string]string{"message": "Game berhasil dihapus!"})
	}	
}

func serveStaticFile(w http.ResponseWriter, r *http.Request, baseDir string, fileServer http.Handler) {
    path := strings.TrimPrefix(r.URL.Path, "/")
    fullPath := filepath.Join(baseDir, path)

    if strings.HasPrefix(r.URL.Path, "/api/") {
        http.NotFound(w, r)
        return
    }

    _, err := os.Stat(fullPath)

    if err != nil {
        // Redirect to "/" on error
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
    }
	fileServer.ServeHTTP(w, r)
}


func main() {
    PORT := 8081
    fileServer := http.FileServer(http.Dir("catalog"))
    connectDB()
    defer db.Close()

    mux := http.NewServeMux()

    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        serveStaticFile(w, r, "catalog", fileServer)
    })

    mux.HandleFunc("/api/games/", handleGames)
    mux.HandleFunc("/api/games", handleGames)

    loggedMux := logRequestHandler(mux)

    fmt.Printf("Server berjalan di http://localhost:%d\n", PORT)
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), loggedMux))
}

// Logging middleware
func logRequestHandler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("Incoming request: %s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}
