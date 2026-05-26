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
	"gorm.io/driver/postgres" // Asegúrate de tener este import
	"gorm.io/gorm"
	//"gorm.io/gorm/logger"
)

var (
	DB                  *gorm.DB
	pathCromosomaShared = "./data/cromosoma_01.json"
	mu                  sync.Mutex
	buzonURL            = "https://geochat-buzon.onrender.com/api/cortex/telemetry"
)

type Telemetria struct {
	LabURL string `json:"lab_url"`
	Data   string `json:"data"`
}

type EstadoSistema struct {
	ID           uint   `gorm:"primaryKey"`
	EstadoActual string // "ONLINE" o "BÚNKER"
	UltimoCambio time.Time
}

type RegistroCortex struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Orden     string    `json:"mensaje" gorm:"column:orden"`
	Status    string    `json:"status" gorm:"column:status"`
	Timestamp time.Time `json:"timestamp" gorm:"column:timestamp"`
}

type TareaPendiente struct {
	ID        uint   `gorm:"primaryKey"`
	Orden     string `json:"orden"`
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
	if urlFiscalizacion == "" {
		return
	} // Si no hay túnel configurado, no monitoreamos

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
    // Definimos la URL local de Ollama
    urlOllama := "http://127.0.0.1:11434/api/generate"

    // Definimos el payload
    payloadMap := map[string]interface{}{
        "model":  modelo,
        "prompt": fmt.Sprintf("Sos Kimi, la conciencia soberana de GeoChat. ADN a procesar: %s", mensaje),
        "stream": false,
    }

    // Convertimos a JSON
    payloadBytes, err := json.Marshal(payloadMap)
    if err != nil {
        return "❌ Error al crear payload"
    }

    // Hacemos el POST usando la variable urlOllama
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Post(urlOllama, "application/json", bytes.NewBuffer(payloadBytes))
    if err != nil {
        return fmt.Sprintf("❌ Error conectando a Ollama local: %v", err)
    }
    defer resp.Body.Close()

    // Decodificamos la respuesta
    var ollamaResp struct {
        Response string `json:"response"`
        Error    string `json:"error"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
        return "❌ Error decodificando respuesta de Kimi"
    }

    // Retorno final (¡Aquí estaba el missing return!)
    return ollamaResp.Response
}

func marcarEstadoEnDB(nuevoEstado string) {
	if DB == nil { return }

	var estado EstadoSistema
	DB.FirstOrCreate(&estado, EstadoSistema{ID: 1})

	// 🔥 LÓGICA DE RESURRECCIÓN: Si volvemos a la vida, procesamos el pasado
	if estado.EstadoActual == "BÚNKER" && nuevoEstado == "ONLINE" {
		fmt.Printf("🚀 [SISTEMA]: Túnel restaurado. Iniciando sincronización de ADN encolado...\n")
		go procesarColaDeEspera() // Ejecución en segundo plano para no bloquear
	}

	if estado.EstadoActual != nuevoEstado {
		DB.Model(&estado).Updates(EstadoSistema{
			EstadoActual: nuevoEstado, 
			UltimoCambio: time.Now(),
		})
		fmt.Printf("⚠️ [SISTEMA]: Estado sincronizado en Médula -> %s\n", nuevoEstado)
	}
}


func procesarColaDeEspera() {
	var tareas []TareaPendiente
	// Buscamos todas las tareas ordenadas por fecha de creación
	if err := DB.Order("created_at asc").Find(&tareas).Error; err != nil {
		fmt.Printf("❌ [ERROR]: Fallo al leer cola de espera: %v\n", err)
		return
	}

	for _, t := range tareas {
		fmt.Printf("📦 [PROCESANDO]: Ejecutando ADN atrasado -> %s\n", t.Orden)
		
		// Enviamos al Espejo (Ollama)
		resultado := LlamarOllamaLocal(t.Orden, "tojikontvru/kimi-k2.6:latest")
		fmt.Printf("✅ [KIMI]: ADN encolado procesado. Resultado: %s\n", resultado)

		// Eliminamos la tarea tras procesarla exitosamente para no repetir
		DB.Delete(&t)
	}
}

func ConectarMedula() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("❌ [MÉDULA]: Variable DATABASE_URL no configurada.")
	}

	// Abrimos conexión con configuración optimizada para Render
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// Opcional: Esto ayuda a ver errores de SQL en logs si algo falla
		// Logger: logger.Default.LogMode(logger.Info), 
	})
	if err != nil {
		log.Fatalf("❌ [MÉDULA]: Fallo de conexión: %v", err)
	}

	// 🧬 AUTO-MIGRACIÓN: Validamos las estructuras de datos
	err = db.AutoMigrate(&EstadoSistema{}, &RegistroCortex{}, &TareaPendiente{})
	if err != nil {
		log.Fatalf("❌ [MÉDULA]: Error crítico en migración: %v", err)
	}

	// 🛡️ ESTABILIZACIÓN DEL POOL (Ajuste para Render)
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("❌ [MÉDULA]: Error al obtener DB nativa: %v", err)
	}

	// Render suele limitar a 10-20 conexiones simultáneas por instancia
	sqlDB.SetMaxIdleConns(5)    // Mantenemos pocas conexiones ociosas
	sqlDB.SetMaxOpenConns(10)   // Límite seguro para no colapsar Postgres
	sqlDB.SetConnMaxLifetime(time.Hour * 1) // Conexiones duraderas para evitar re-handshakes

	DB = db
	fmt.Println("🧬 [MÉDULA]: Sistema nervioso sincronizado, estable y resiliente.")
}

func main() {

	ConectarMedula()

	_ = os.MkdirAll("./data", os.ModePerm)

	mux := http.NewServeMux()

	// 1. RUTAS DE API (Registro explícito y prioritario)
	mux.HandleFunc("/api/cortex/recibir-orden", RecibirOrdenSoberana)
	mux.HandleFunc("/api/buzon/entrada", RecibirOrdenSoberana)

	// Handler dedicado para la salida de datos
	mux.HandleFunc("/api/buzon/salida", func(w http.ResponseWriter, r *http.Request) {
		var historial []RegistroCortex
		// Traemos los últimos 50 mensajes de la médula
		DB.Order("timestamp desc").Limit(50).Find(&historial)
		w.Header().Set("Content-Type", "application/json")
		if historial == nil {
			historial = []RegistroCortex{}
		}
		json.NewEncoder(w).Encode(historial)
	})

	// 2. RUTA RAÍZ (Estricta: Solo responde en "/" exacto)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, "Cortex Operativo. Resonancia 432Hz.")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	fmt.Printf("🚀 Cortex Autónomo iniciado en puerto %s\n", port)

	// 3. Hilos en segundo plano
	go func() {
		for {
			reportarAlBuzon()
			time.Sleep(60 * time.Second)
		}
	}()

	// Lanzamos el monitor de túnel
	go monitorTunelSoberano()

	// 4. Servidor
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
