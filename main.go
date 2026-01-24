package main

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	"strings"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Price struct {
	ID         int
	Name       string
	Category   string
	Price      float64
	CreateDate string
}

type PostResponse struct {
	TotalItems      int     `json:"total_items"`
	TotalCategories int     `json:"total_categories"`
	TotalPrice      float64 `json:"total_price"`
}

var db *sql.DB

func main() {
	var err error
	db, err = connectDB()
	if err != nil {
		log.Println("Error:", err)
	}
	defer db.Close()
    log.Println("DB connected")
	router := mux.NewRouter()
	router.HandleFunc("/api/v0/prices", handlePostPrices).Methods("POST")
	router.HandleFunc("/api/v0/prices", handleGetPrices).Methods("GET")
	port := ":8080"
	log.Printf("HTTP server started on port %s", port)
	if err := http.ListenAndServe(port, router); err != nil {
		log.Println("Error", err)
	}
}

func connectDB() (*sql.DB, error) {
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	database, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err = database.Ping(); err != nil {
		return nil, err
	}
	return database, nil
}

func insertPriceData(prices []Price) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare("INSERT INTO prices (name, category, price, create_date) VALUES ($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, price := range prices {
		_, err := stmt.Exec(price.Name, price.Category, price.Price, price.CreateDate)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func getAllPrices() ([]Price, error) {
	rows, err := db.Query("SELECT id, name, category, price, create_date FROM prices ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []Price
	for rows.Next() {
		var price Price
		var createDate time.Time
		err := rows.Scan(&price.ID, &price.Name, &price.Category, &price.Price, &createDate)
		if err != nil {
			return nil, err
		}
		price.CreateDate = createDate.Format("2006-01-02")
		prices = append(prices, price)
	}

	return prices, rows.Err()
}

func getStatistics() (*PostResponse, error) {
	var response PostResponse

	query := `
		SELECT
			COUNT(*) as total_items,
			COUNT(DISTINCT category) as total_categories,
			COALESCE(SUM(price), 0) as total_price
		FROM prices
	`

	err := db.QueryRow(query).Scan(&response.TotalItems, &response.TotalCategories, &response.TotalPrice)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func handlePostPrices(w http.ResponseWriter, r *http.Request) {
	log.Println("Получен POST запрос на /api/v0/prices")
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	csvData, err := unzipFile(fileBytes)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	prices, err := parseCSV(csvData)
	if err != nil {
		http.Error(w, "Ошибка парсинга CSV: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := insertPriceData(prices); err != nil {
		http.Error(w, "Ошибка вставки в БД: "+err.Error(), http.StatusInternalServerError)
		return
	}
	stats, err := getStatistics()
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func handleGetPrices(w http.ResponseWriter, r *http.Request) {
	prices, err := getAllPrices()
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	csvData, err := writeCSV(prices)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	zipData, err := createZipArchive(csvData)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=data.zip")
	w.Write(zipData)
}

func unzipFile(data []byte) ([]byte, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if strings.HasSuffix(file.Name, "data.csv") {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			
			csvData, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}
			log.Printf("file data.csv found: %s", file.Name)
			return csvData, nil
		}
	}

	return nil, fmt.Errorf("file data.csv not found")
}

func parseCSV(data []byte) ([]Price, error) {
	reader := csv.NewReader(bytes.NewReader(data))

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 1 {
		return nil, fmt.Errorf("CSV file empty")
	}
	var prices []Price
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 5 {
			continue
		}
		id, err := strconv.Atoi(record[0])
		if err != nil {
			continue
		}
		price, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			continue
		}
		prices = append(prices, Price{
			ID:         id,
			Name:       record[1],
			Category:   record[2],
			Price:      price,
			CreateDate: record[4],
		})
	}

	return prices, nil
}

func writeCSV(prices []Price) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	header := []string{"id", "name", "category", "price", "create_date"}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	for _, price := range prices {
		record := []string{
			strconv.Itoa(price.ID),
			price.Name,
			price.Category,
			fmt.Sprintf("%.2f", price.Price),
			price.CreateDate,
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func createZipArchive(csvData []byte) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	fileWriter, err := zipWriter.Create("data.csv")
	if err != nil {
		return nil, err
	}
	if _, err := fileWriter.Write(csvData); err != nil {
		return nil, err
	}
	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
