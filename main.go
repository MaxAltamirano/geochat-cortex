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
		resp, err := http.Get("http://localhost:10000")
		if err != nil {
			fmt.Println("🛰️ [ALERTA]: Túnel caído. Auto-reparando...")
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

func LlamarProxyGemini(mensaje string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey

	payloadMap := map[string]interface{}{
		"contents": []map[string]interface{}{{"parts": []map[string]interface{}{{"text": mensaje}}}},
	}
	payloadBytes, _ := json.Marshal(payloadMap)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("error de red: %v", err)
	}
	defer resp.Body.Close()

	// 🛡️ AQUÍ ESTÁ EL FILTRO SOBERANO:
	// Si Google responde con 403, 400, 500, etc., no intentamos parsear nada.
	if resp.StatusCode != http.StatusOK {
		bodyError, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("nexo denegado (Status %d): %s", resp.StatusCode, string(bodyError))
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	var res map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return "", fmt.Errorf("error al decodificar JSON: %v", err)
	}

	// 🛡️ Validaciones de seguridad para evitar "panic" si la estructura cambia
	candidates, ok := res["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "", fmt.Errorf("estructura de respuesta inválida: no hay candidatos")
	}

	content, ok := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("estructura de respuesta inválida: no hay content")
	}

	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return "", fmt.Errorf("estructura de respuesta inválida: no hay partes")
	}

	text, ok := parts[0].(map[string]interface{})["text"].(string)
	if !ok {
		return "", fmt.Errorf("no se pudo extraer el texto")
	}

	return text, nil
}

func main() {
	_ = os.MkdirAll("./data", os.ModePerm)

	// Hilo 1: Telemetría ajustada a 60 segundos
	go func() {
		for {
			reportarAlBuzon()
			time.Sleep(60 * time.Second)
		}
	}()

	// Hilo 2: Guardián de Conectividad
	go monitorTunelSoberano()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)

			// 1. Persistencia Inmediata
			mu.Lock()
			os.WriteFile(pathCromosomaShared, body, 0644)
			mu.Unlock()

			fmt.Println("🧬 ADN RECIBIDO Y PERSISTIDO")

			// 2. Sistema Nervioso: Activación de IA en segundo plano
			var msg struct {
				Orden string `json:"orden"`
			}
			if err := json.Unmarshal(body, &msg); err == nil && msg.Orden != "" {
				go func(orden string) {
					respuesta, err := LlamarProxyGemini(orden)
					if err != nil {
						fmt.Printf("❌ Error IA: %v\n", err)
					} else {
						fmt.Printf("🤖 [GEMINI]: %s\n", respuesta)

						// NUEVO: Guardar la respuesta para que el Front la vea
						mu.Lock()
						os.WriteFile("./data/respuesta_ia.json", []byte(respuesta), 0644)
						mu.Unlock()
					}
				}(msg.Orden)
			}

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

	fmt.Printf("🚀 Cortex Autónomo iniciado en puerto %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
