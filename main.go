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

const (
	SUCCESS = "\033[1;32m"
	GOLD    = "\033[1;33m"
	NEON    = "\033[1;36m"
	ERROR   = "\033[1;31m"
	NC      = "\033[0m"
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


type Task struct {
	ID          uint   `gorm:"primaryKey"`
	Titulo      string `gorm:"column:titulo"`
	Descripcion string `gorm:"column:descripcion"`
	Estado      string `gorm:"column:estado;default:pendiente"`
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
        
        // 2. FILOSOFÍA DE ESPEJO PASIVO:
        // Render NO procesa IA. Render solo encola. 
        // El Nodo Local (Linux) es el único que tiene permiso para ejecutar Ollama.
        
        fmt.Printf("📥 [CORTEX]: ADN recibido. Encolando para procesamiento soberano local: %s\n", msg.Orden)
        
        // Guardamos en la médula (Postgres) para que el Nodo Local lo vea y ejecute
        guardarEnColaDeEspera(msg.Orden)

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusAccepted) // 202 Accepted: El sistema lo tomó, pero el proceso es asíncrono
        json.NewEncoder(w).Encode(map[string]string{
            "status": "ADN_ENCOLADO",
            "info":   "El Nodo Local procesará esta orden en el próximo pulso.",
        })
        return
    }

    http.Error(w, "ADN inválido", http.StatusBadRequest)
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
		go ProcesarColaPendiente() // Ejecución en segundo plano para no bloquear
	}

	if estado.EstadoActual != nuevoEstado {
		DB.Model(&estado).Updates(EstadoSistema{
			EstadoActual: nuevoEstado, 
			UltimoCambio: time.Now(),
		})
		fmt.Printf("⚠️ [SISTEMA]: Estado sincronizado en Médula -> %s\n", nuevoEstado)
	}
}


// En tu función de procesamiento local, añade un "Watchdog" de RAM
func ProcesarColaPendiente() {
    var tareas []Task
    
    // Usamos el resultado para validar si hay carga de trabajo real
    result := DB.Where("estado = ?", "pendiente").Find(&tareas)
    
    // 1. Verificamos si hubo error en la consulta
    if result.Error != nil {
        log.Printf("❌ [MÉDULA]: Error consultando cola: %v", result.Error)
        return
    }

    // 2. Verificamos si hay tareas usando el resultado (RowsAffected)
    if result.RowsAffected == 0 {
        // No hay nada que procesar, salimos limpiamente
        return 
    }

    fmt.Printf("📋 [CÓRTEX]: Procesando %d tareas pendientes...\n", result.RowsAffected)

    for _, tarea := range tareas {
        err := EnviarARender(tarea)
        if err == nil {
            // Actualizamos el estado solo si el envío fue exitoso
            DB.Model(&tarea).Update("estado", "sincronizado")
        } else {
            log.Printf("⚠️ [CÓRTEX]: Tarea %d fallida, reintentando en el próximo pulso.", tarea.ID)
            // No hacemos nada más, dejamos que la tarea siga pendiente
        }
    }
}


// --- 📡 NEXO DE COMUNICACIÓN CON RENDER ---
func EnviarARender(tarea Task) error {
	// URL hacia tu API en Render
	url := "https://geochat-buzon.onrender.com/api/cortex/recibir-orden"

	// Estructura del ADN a transmitir
	data := map[string]interface{}{
		"id_adn":           "MAIN-LAB-SYNC", // O tu variable global memoriaADN.DNA_ID
		"orden":            tarea.Titulo + ": " + tarea.Descripcion,
		"contexto_tecnico": "Sincronización Médula-Render",
		"target_ia":        "KIMI",
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error serializando ADN: %v", err)
	}

	// Disparo de red
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf(ERROR+"❌ Fallo de conexión con Render: %v"+NC, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error en Render: código %d", resp.StatusCode)
	}

	fmt.Printf("%s📡 [NEXO]: ADN enviado a Render con éxito.%s\n", SUCCESS, NC)
	return nil
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
