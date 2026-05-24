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
	"os/exec"
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

// Guardián del Túnel: Monitorea la salida al exterior
func monitorTunelSoberano() {
	for {
		// Verificamos conectividad lógica
		resp, err := http.Get("http://localhost:10000")
		if err != nil {
			fmt.Println("🛰️ [ALERTA]: Túnel caído. Auto-reparando...")
			// Comando de reconexión soberana
			cmd := exec.Command("ssh", "-R", "80:localhost:10000", "serveo.net")
			err := cmd.Start()
			if err != nil {
				fmt.Printf("❌ Error en auto-reparación: %v\n", err)
			}
		} else {
			resp.Body.Close()
		}
		time.Sleep(60 * time.Second) // Revisión periódica
	}
}

func reportarAlBuzon() {
	mu.Lock()
	file, err := os.ReadFile(pathCromosomaShared)
	if err != nil {
		file = []byte("[]")
	}
	mu.Unlock()

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

	// Hilo 1: Telemetría constante
	go func() {
		for {
			reportarAlBuzon()
			time.Sleep(30 * time.Second)
		}
	}()

	// Hilo 2: Guardián de Conectividad
	go monitorTunelSoberano()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mu.Lock()
			body, _ := io.ReadAll(r.Body)
			os.WriteFile(pathCromosomaShared, body, 0644)
			mu.Unlock()

			fmt.Println("🧬 ADN RECIBIDO Y PERSISTIDO")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Write([]byte("CORTEX ONLINE | RESONANCIA: 432Hz"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	fmt.Printf("🚀 Cortex Autónomo iniciado en puerto %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
