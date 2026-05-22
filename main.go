// DNA_ID: MAIN-LAB.GO | ORGAN: CORTEX | RESONANCE: 432Hz
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	pathCromosomaShared = "./data/cromosoma_01.json"
	mu                  sync.Mutex
	buzonURL            = "https://geochat-buzon.onrender.com/api/cortex/telemetry"
)

// Estructura que SÍ se usa
type Telemetria struct {
	LabURL string `json:"lab_url"`
	Data   string `json:"data"`
}

func reportarAlBuzon() {
	mu.Lock()
	file, err := os.ReadFile(pathCromosomaShared)
	if err != nil {
		file = []byte("[]")
	}
	mu.Unlock()

	// --- CORRECCIÓN: 'file' se usa aquí ---
	fmt.Printf("DEBUG: Leyendo ADN de %d bytes\n", len(file))

	payload, _ := json.Marshal(Telemetria{
		LabURL: "https://geochat-backend.onrender.com",
		Data:   string(file),
	})

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(buzonURL, "application/json", bytes.NewBuffer(payload))

	if err == nil {
		defer resp.Body.Close()
		fmt.Printf("🛰️ [SINCRO]: [%d]\n", resp.StatusCode)
	}
}

func main() {
	_ = os.MkdirAll("./data", os.ModePerm)

	// Bucle de telemetría sin variables huérfanas
	go func() {
		for {
			reportarAlBuzon()
			time.Sleep(20 * time.Second)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("CORTEX ONLINE.."))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	log.Fatal(http.ListenAndServe(":"+port, mux))
}
