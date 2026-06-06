package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Estructura de Médula para persistencia
type TareaIA struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Orden     string    `json:"orden"`
	Respuesta string    `json:"respuesta"`
	Estado    string    `json:"estado"`
	Logs      string    `json:"logs"`
	Timestamp time.Time `json:"timestamp"`
}

// Estructura para Grafo (Simplificado para navegación de módulos)
type NodoModulo struct {
	Nombre       string
	Estado       string
	Dependencias []*NodoModulo
}

type GrafoCortex struct {
	Raiz *NodoModulo
	sync.RWMutex
}

// Cache Dinámica en RAM
var CortexCache = struct {
	ADN_Maestro map[string]interface{}
	Trilogia    string
	Grafo       *GrafoCortex
	sync.RWMutex
}{
	ADN_Maestro: make(map[string]interface{}),
	Grafo:       &GrafoCortex{Raiz: &NodoModulo{Nombre: "ROOT"}},
}

var DB *gorm.DB

func main() {
	dsn := os.Getenv("DATABASE_URL")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Error conectando a DB:", err)
	}
	DB = db
	DB.AutoMigrate(&TareaIA{})

	http.HandleFunc("/api/ordenar", recibirOrden)
	http.HandleFunc("/api/consultar", entregarResultado)
	http.HandleFunc("/api/configurar", configurarNodo) // Punto de inyección de ADN

	log.Printf("🚀 [CÓRTEX]: Online en puerto %s", os.Getenv("PORT"))
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}

// 1. Inyección de ADN (vía despertar.sh)
func configurarNodo(w http.ResponseWriter, r *http.Request) {
	var config struct {
		ADN_Maestro map[string]interface{} `json:"adn_maestro"`
		Trilogia    string                 `json:"trilogia"`
	}
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	CortexCache.Lock()
	CortexCache.ADN_Maestro = config.ADN_Maestro
	CortexCache.Trilogia = config.Trilogia
	CortexCache.Unlock()

	log.Printf("✨ [CÓRTEX]: ADN inyectado. Consciencia cargada.")
	w.WriteHeader(http.StatusOK)
}

// 2. Procesamiento Inteligente
func recibirOrden(w http.ResponseWriter, r *http.Request) {
	var t TareaIA
	json.NewDecoder(r.Body).Decode(&t)
	t.Estado = "PENDIENTE"
	DB.Create(&t)
	go procesarConIA(&t)
	w.WriteHeader(http.StatusAccepted)
}

func procesarConIA(t *TareaIA) {
	// Acceso rápido a Identidad
	CortexCache.RLock()
	adn := CortexCache.ADN_Maestro
	trilogia := CortexCache.Trilogia
	CortexCache.RUnlock()

	log.Printf("🧠 [CÓRTEX]: Iniciando tarea con ADN: %v | Regla: %s", adn["nodo_id"], trilogia)

	// Aquí llamarías a la IA pasando el ADN como contexto del sistema
	ejecutarModularizacion(t, "modulo_comunitario_gps", "package main\nimport \"fmt\"\nfunc main() { fmt.Println('GPS activo') }")
}

func ejecutarModularizacion(t *TareaIA, nombre string, codigo string) {
	dir := fmt.Sprintf("./sandbox/%d/%s", t.ID, nombre)
	os.MkdirAll(dir, 0755)
	ruta := fmt.Sprintf("%s/main.go", dir)
	os.WriteFile(ruta, []byte(codigo), 0644)

	cmd := exec.Command("go", "build", "-o", "binario", ruta)
	var out bytes.Buffer
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		DB.Model(&t).Updates(TareaIA{Logs: out.String(), Estado: "ERROR_COMPILACION"})
	} else {
		DB.Model(&t).Update("Estado", "PROCESADO")
	}
}

func entregarResultado(w http.ResponseWriter, r *http.Request) {
	var completadas []TareaIA
	DB.Where("estado = ?", "PROCESADO").Find(&completadas)
	json.NewEncoder(w).Encode(completadas)
}
