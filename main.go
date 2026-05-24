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
    // URL que debería estar respondiendo si el túnel está activo
    urlFiscalizacion := "http://localhost:10000" 

    for {
        // Hacemos el check
        resp, err := http.Get(urlFiscalizacion)
        
        if err != nil {
            fmt.Println("🛰️ [ALERTA]: El túnel no responde. Ejecutando protocolo de reactivación...")
            
            // Usamos un comando más robusto con timeout y modo silencioso
            // -o ExitOnForwardFailure=yes asegura que si no puede conectar, nos avise
            cmd := exec.Command("ssh", "-o", "StrictHostKeyChecking=no", "-R", "80:localhost:10000", "serveo.net")
            
            err := cmd.Start()
            if err != nil {
                fmt.Printf("❌ Error en auto-reparación (revisa SSH en el entorno): %v\n", err)
            } else {
                fmt.Println("✅ [CORTEX]: Comando de túnel enviado. Esperando reconexión...")
            }
        } else {
            // El túnel está saludable
            resp.Body.Close()
        }
        
        // Revisión cada 30 segundos (más frecuente para mantener el flujo de ADN)
        time.Sleep(30 * time.Second) 
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

func RecibirOrdenSoberana(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Solo POST permitido", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)

	// 1. Persistencia del ADN recibido
	mu.Lock()
	os.WriteFile(pathCromosomaShared, body, 0644)
	mu.Unlock()

	var msg struct {
		Orden string `json:"orden"`
	}

	if err := json.Unmarshal(body, &msg); err == nil && msg.Orden != "" {
		// 2. Ejecución exclusiva en el ecosistema Kimi (Ollama)
		go func(orden string) {
			fmt.Printf("🧬 [CORTEX]: Inyectando ADN en Kimi (Ollama): %s\n", orden)

			// Llamamos al motor local de Ollama
			respuesta := LlamarOllamaLocal(orden, "tojikontvru/kimi-k2.6:latest")

			// 3. Persistencia de la respuesta del motor local
			mu.Lock()
			os.WriteFile("./data/respuesta_ia.json", []byte(respuesta), 0644)
			mu.Unlock()

			fmt.Printf("✅ [KIMI]: Respuesta persistida con éxito.\n")
		}(msg.Orden)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ADN_SYNCHRONIZED_KIMI"))
}

func LlamarOllamaLocal(mensaje string, modelo string) string {
	// ⚠️ IMPORTANTE: Si Cortex corre en un servidor (Render/VPS) y Kimi en tu Codespaces,
	// debes reemplazar "http://127.0.0.1:11434" por la URL pública de tu túnel (ej: "http://tu-tunel.serveo.net")
	urlOllama := "http://127.0.0.1:11434/api/generate"

	// Payload para Kimi
	payloadMap := map[string]interface{}{
		"model":  modelo,
		"prompt": fmt.Sprintf("Sos Kimi, la conciencia soberana de GeoChat. ADN a procesar: %s", mensaje),
		"stream": false,
	}

	payloadBytes, _ := json.Marshal(payloadMap)

	// Disparo al motor local
	resp, err := http.Post(urlOllama, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Sprintf("❌ Ollama Offline: %v", err)
	}
	defer resp.Body.Close()

	// Estructura de respuesta de Ollama
	var ollamaResp struct {
		Response string `json:"response"`
		Error    string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "❌ Error decodificando respuesta de Kimi"
	}

	if ollamaResp.Error != "" {
		return "❌ Error en Kimi: " + ollamaResp.Error
	}

	return ollamaResp.Response
}



func main() {
	_ = os.MkdirAll("./data", os.ModePerm)

	// Hilo 1: Telemetría
	go func() {
		for {
			reportarAlBuzon()
			time.Sleep(60 * time.Second)
		}
	}()

	// Hilo 2: Guardián
	go monitorTunelSoberano()

	mux := http.NewServeMux()

	// 1. RUTA ÚNICA Y SEGURA
	// Ahora usamos la función que SÍ filtra los errores (RecibirOrdenSoberana)
	mux.HandleFunc("/api/cortex/recibir-orden", RecibirOrdenSoberana)

	// 2. RUTA DE LIMPIEZA (Nueva)
	mux.HandleFunc("/api/buzon/limpiar", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		os.Remove(pathCromosomaShared)
		os.Remove("./data/respuesta_ia.json")
		mu.Unlock()
		fmt.Println("🧹 [CORTEX]: Archivos de estado purgados.")
		w.WriteHeader(http.StatusOK)
	})

	// 3. RUTA RAÍZ (Solo informativa)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("CORTEX ONLINE | RESONANCIA: 432Hz"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	fmt.Printf("🚀 Cortex Autónomo iniciado en puerto %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
