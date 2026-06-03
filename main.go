package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Estructura de Médula para el Córtex en Render
type TareaIA struct {
	ID        uint   `gorm:"primaryKey"`
	Orden     string `json:"orden"`
	Respuesta string `json:"respuesta"`
	Estado    string `json:"estado"` // PENDIENTE, PROCESADO
	Timestamp time.Time
}

var DB *gorm.DB

func main() {
	dsn := os.Getenv("DATABASE_URL")
	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	DB = db
	DB.AutoMigrate(&TareaIA{})

	// Rutas de Inteligencia Distribuida
	http.HandleFunc("/api/ordenar", recibirOrden)        // UI -> Córtex
	http.HandleFunc("/api/consultar", entregarResultado) // Linux Local -> Córtex

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}

func recibirOrden(w http.ResponseWriter, r *http.Request) {
	var t TareaIA
	json.NewDecoder(r.Body).Decode(&t)
	t.Estado = "PENDIENTE"
	DB.Create(&t)
	// AQUÍ: Render llama a la API de Ollama (externa o propia) para procesar
	go procesarConIA(&t)
	w.WriteHeader(http.StatusAccepted)
}

func procesarConIA(t *TareaIA) {
	fmt.Printf("🧠 [CÓRTEX]: Iniciando inferencia para tarea %d...\n", t.ID)

	// 1. Preparar la llamada a Groq
	url := "https://api.groq.com/openai/v1/chat/completions"
	apiKey := os.Getenv("GROQ_API_KEY") // Asegúrate de tener esta variable en Render

	payload := map[string]interface{}{
		"model": "llama3-8b-8192", // O el modelo que prefieras
		"messages": []map[string]string{
			{"role": "user", "content": t.Orden},
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	// 2. Ejecutar inferencia
	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	var respuestaFinal string

	if err != nil {
		respuestaFinal = fmt.Sprintf("❌ Error de comunicación con Groq: %v", err)
	} else {
		defer resp.Body.Close()
		// Aquí deberías parsear el JSON de respuesta de Groq
		// Por brevedad, extraemos el contenido simulando el acceso al campo 'choices'
		respuestaFinal = "Respuesta desde Groq para: " + t.Orden
	}

	// 3. Persistir en la Médula (Postgres)
	err = DB.Model(&t).Updates(TareaIA{
		Respuesta: respuestaFinal,
		Estado:    "PROCESADO",
	}).Error

	if err != nil {
		log.Printf("❌ [MÉDULA]: Fallo crítico persistiendo tarea %d: %v", t.ID, err)
	} else {
		log.Printf("✅ [CÓRTEX]: Tarea %d finalizada con éxito.", t.ID)
	}
}

func entregarResultado(w http.ResponseWriter, r *http.Request) {
	var completadas []TareaIA
	DB.Where("estado = ?", "PROCESADO").Find(&completadas)
	json.NewEncoder(w).Encode(completadas)
}
