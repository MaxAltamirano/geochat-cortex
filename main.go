// DNA_ID: MAIN-LAB.GO | ORGAN: CORTEX | RESONANCE: 432Hz
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	// Asegurar persistencia de datos
	_ = os.MkdirAll("./data", os.ModePerm)

	// Bucle de telemetría soberana
	go func() {
		for {
			reportarAlBuzon()
			time.Sleep(20 * time.Second)
		}
	}()

	mux := http.NewServeMux()

	// Handler Central para el ADN y Órdenes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mu.Lock()
			defer mu.Unlock()

			// Leer el ADN entrante desde el body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error leyendo ADN", http.StatusInternalServerError)
				return
			}

			// Escribir ADN en el cromosoma permanente
			err = os.WriteFile(pathCromosomaShared, body, 0644)
			if err != nil {
				fmt.Printf("❌ Error grabando ADN: %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			fmt.Println("🧬 ADN RECIBIDO Y PERSISTIDO EN MÉDULA")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ADN_SYNCHRONIZED"))
			return
		}
		w.Write([]byte("CORTEX ONLINE | RESONANCIA: 432Hz"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	fmt.Printf("🚀 Cortex iniciado en puerto %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
