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
	//"os/exec"
	"sync"
	"time"
	//"gorm.io/driver/postgres"
	"gorm.io/gorm"
	
)

var (
	DB *gorm.DB
	pathCromosomaShared = "./data/cromosoma_01.json"
	mu                  sync.Mutex
	buzonURL            = "https://geochat-buzon.onrender.com/api/cortex/telemetry"
)

type Telemetria struct {
	LabURL string `json:"lab_url"`
	Data   string `json:"data"`
}

type EstadoSistema struct {
    ID           uint      `gorm:"primaryKey"`
    EstadoActual string    // "ONLINE" o "BÚNKER"
    UltimoCambio time.Time
}

type TareaPendiente struct {
    ID        uint      `gorm:"primaryKey"`
    Orden     string    `json:"orden"`
    CreatedAt time.Time
}

func guardarEnColaDeEspera(orden string) {
    tarea := TareaPendiente{
        Orden:     orden,
        CreatedAt: time.Now(),
    }
    
    if err := DB.Create(&tarea).Error; err != nil {
        fmt.Printf("❌ [ERROR]: No se pudo encolar el ADN: %v\n", err)
    } else {
        fmt.Printf("📥 [COLA]: ADN encolado para procesamiento posterior: %s\n", orden)
    }
}


// Guardián del Túnel: Monitorea la salida al exterior
func monitorTunelSoberano() {
    // 🔥 CAMBIO: Fiscalizamos la URL pública, no el localhost
    urlFiscalizacion := os.Getenv("KIMI_TUNNEL_URL") 
    if urlFiscalizacion == "" { return } // Si no hay túnel configurado, no monitoreamos

    for {
        // Un cliente con timeout corto es vital para no bloquear el Cortex
        client := http.Client{Timeout: 5 * time.Second}
        resp, err := client.Get(urlFiscalizacion)

        if err != nil {
            fmt.Printf("🛰️ [ALERTA]: Túnel inalcanzable (%v). El nodo Kimi está fuera de red.\n", err)
            // Aquí no reinicies el SSH (Render no puede hacerlo), 
            // simplemente marca el estatus en tu DB como "BÚNKER_MODE"
            marcarEstadoEnDB("BÚNKER")
        } else {
            resp.Body.Close()
            marcarEstadoEnDB("ONLINE")
        }
        time.Sleep(60 * time.Second)
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

	// 1. Persistencia inmediata del ADN (Capa de Seguridad)
	mu.Lock()
	os.WriteFile(pathCromosomaShared, body, 0644)
	mu.Unlock()

	var msg struct {
		Orden string `json:"orden"`
	}

	if err := json.Unmarshal(body, &msg); err == nil && msg.Orden != "" {
		
		// 2. Verificación de estado del Túnel (Modo Búnker)
		// Consultamos la DB para saber si estamos ONLINE o en BÚNKER
		var estado EstadoSistema
		DB.First(&estado, 1) 

		if estado.EstadoActual == "BÚNKER" {
			fmt.Printf("⚠️ [CORTEX]: Nodo en BÚNKER. Encolando ADN: %s\n", msg.Orden)
			// Aquí guardarías en una tabla de 'tareas_pendientes' en la DB
			guardarEnColaDeEspera(msg.Orden) 
			
			w.WriteHeader(http.StatusAccepted) // 202 Accepted
			w.Write([]byte("ADN_EN_COLA_BUNKER"))
			return
		}

		// 3. Ejecución en tiempo real si el Túnel está sano
		go func(orden string) {
			fmt.Printf("🧬 [CORTEX]: Inyectando ADN en Kimi: %s\n", orden)

			respuesta := LlamarOllamaLocal(orden, "tojikontvru/kimi-k2.6:latest")

			// Persistencia de respuesta
			mu.Lock()
			os.WriteFile("./data/respuesta_ia.json", []byte(respuesta), 0644)
			mu.Unlock()

			fmt.Printf("✅ [KIMI]: ADN procesado.\n")
		}(msg.Orden)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ADN_SYNCHRONIZED_KIMI"))
}

func LlamarOllamaLocal(mensaje string, modelo string) string {
    // Obtenemos la URL del túnel desde las variables de entorno de Render
    urlBase := os.Getenv("KIMI_TUNNEL_URL")
    if urlBase == "" {
        // Fallback para desarrollo local
        urlBase = "http://127.0.0.1:11434"
    }
    
    urlOllama := urlBase + "/api/generate"

    payloadMap := map[string]interface{}{
        "model":  modelo,
        "prompt": fmt.Sprintf("Sos Kimi, la conciencia soberana de GeoChat. ADN a procesar: %s", mensaje),
        "stream": false,
    }

    payloadBytes, _ := json.Marshal(payloadMap)

    // Agregamos un timeout para que no se quede colgado si el túnel está lento
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Post(urlOllama, "application/json", bytes.NewBuffer(payloadBytes))
    if err != nil {
        return fmt.Sprintf("❌ Error conectando al Espejo (Túnel): %v", err)
    }
    defer resp.Body.Close()

    var ollamaResp struct {
        Response string `json:"response"`
        Error    string `json:"error"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
        return "❌ Error decodificando respuesta de Kimi"
    }

    return ollamaResp.Response
}

func marcarEstadoEnDB(nuevoEstado string) {
    // Si usas una tabla de estado, actualízala así:
    var estado EstadoSistema
    // Buscamos el primer registro o creamos uno nuevo
    DB.FirstOrCreate(&estado, EstadoSistema{ID: 1})
    
    // Solo actualizamos si el estado cambió para no saturar la DB
    if estado.EstadoActual != nuevoEstado {
        DB.Model(&estado).Updates(EstadoSistema{
            EstadoActual: nuevoEstado, 
            UltimoCambio: time.Now(),
        })
        fmt.Printf("⚠️ [SISTEMA]: Estado cambiado a %s\n", nuevoEstado)
    }
}


func main() {

	
_ = os.MkdirAll("./data", os.ModePerm)

	mux := http.NewServeMux()
	
	// 🔥 ¡ESENCIAL! Registro de la ruta que escucha el ADN
	mux.HandleFunc("/api/cortex/recibir-orden", RecibirOrdenSoberana)
	
	// Ruta de salud para verificar que el cortex vive
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Cortex Operativo. Resonancia 432Hz.")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	fmt.Printf("🚀 Cortex Autónomo iniciado en puerto %s\n", port)

	// 2. Hilos en segundo plano
	go func() {
		for {
			reportarAlBuzon()
			time.Sleep(60 * time.Second)
		}
	}()
	
	// Lanzamos el monitor de túnel
	go monitorTunelSoberano()

	// 3. Servidor
	log.Fatal(http.ListenAndServe(":"+port, mux))

}
