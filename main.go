package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// Definimos la estructura del "Gemelo Digital"
type GemeloDigital struct {
	DID       string    `json:"did"`
	Timestamp time.Time `json:"timestamp"`
	ADNHash   string    `json:"adn_hash"` // Hash del código fuente actual
	Status    string    `json:"status"`
}

func evolucionarCortex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Solo método POST permitido", http.StatusMethodNotAllowed)
		return
	}

	// 1. Leer el código enviado por Kimi
	nuevoCodigo, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error al leer cuerpo", http.StatusBadRequest)
		return
	}

	// 2. Guardar en el archivo temporal del Sandbox
	err = ioutil.WriteFile("./sandbox/tmp_evolucion.go", nuevoCodigo, 0644)
	if err != nil {
		http.Error(w, "Error al escribir en Sandbox", http.StatusInternalServerError)
		return
	}

	// 3. Ejecutar el script de validación (el que creaste: scripts/test_evolucion.sh)
	cmd := exec.Command("bash", "./scripts/test_evolucion.sh")
	output, err := cmd.CombinedOutput()

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "❌ Evolución fallida:\n%s", string(output))
		return
	}

	// 4. SI PASA EL TEST: Aplicar el cambio permanentemente
	exec.Command("mv", "./sandbox/tmp_evolucion.go", "main.go").Run()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "✅ Evolución aplicada exitosamente. Reiniciando sistema en el próximo ciclo.")
}

func enableCors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			return
		}
		next(w, r)
	}
}

// getHeartbeat entrega el ADN y estado actual de la instancia al Vigilante Local.
func getHeartbeat(w http.ResponseWriter, r *http.Request) {
	// 1. Opcional: Aquí implementarías la verificación de DID para proteger el endpoint
	// if !VerificarAcceso(r) { http.Error(w, "No autorizado", http.StatusUnauthorized); return }

	// 2. Cálculo dinámico del ADN (Hash actual del ejecutable/fuente)
	// Usamos "main.go" porque es el corazón que compilas en tu Sandbox
	hashActual, err := calcularHash("main.go")
	if err != nil {
		log.Printf("⚠️ [CORTEX]: Error crítico calculando ADN: %v", err)
		hashActual = "error_calculando"
	}

	// 3. Construcción del estado del Gemelo Digital
	estado := GemeloDigital{
		DID:       "0x8A3853EB94AbF84eFe0626093A42F2fa9c9Da310",
		Timestamp: time.Now(),
		ADNHash:   hashActual,
		Status:    "OPERATIVO",
	}

	// 4. Respuesta profesional
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(estado); err != nil {
		log.Printf("❌ [CORTEX]: Error al enviar Heartbeat: %v", err)
		http.Error(w, "Error interno de servidor", http.StatusInternalServerError)
		return
	}
}

func calcularHash(archivo string) (string, error) {
	f, err := os.Open(archivo)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func main() {
	// Aplicamos el middleware a la ruta
	http.HandleFunc("/api/cortex/evolucionar", enableCors(evolucionarCortex))
	http.HandleFunc("/api/sync/heartbeat", getHeartbeat)

	http.HandleFunc("/api/sync/parchear", func(w http.ResponseWriter, r *http.Request) {
		// 1. Validar que solo yo pueda parchar
		// (A futuro: añadir firma digital aquí)

		// 2. Leer el nuevo código enviado
		nuevoCodigo, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Fallo al leer parche", http.StatusBadRequest)
			return
		}

		// 3. Sobrescribir el main.go actual
		err = os.WriteFile("main.go", nuevoCodigo, 0644)
		if err != nil {
			http.Error(w, "Error al escribir parche", http.StatusInternalServerError)
			return
		}

		// 4. Salir para que Render fuerce el reinicio (o ejecutar un comando de build)
		log.Println("🚨 [CORTEX]: Parche aplicado. Reiniciando organismo...")
		os.Exit(1)
	})

	http.HandleFunc("/api/cortex/telemetria", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// 1. Calculamos el hash (asegúrate de tener una función calcularHash implementada)
		hashActual, _ := calcularHash("main.go")

		// 2. Telemetría consolidada
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ONLINE",
			"adn_hash":  hashActual,
			"latencia":  time.Now().UnixNano(),
			"ideas":     []map[string]string{}, // Aquí irían tus ideas pendientes
			"buzon_msg": "Nodo GeoChat Activo", // Simulación de lectura de buzón
		})
	})

	fmt.Println("🚀 [CÓRTEX]: Evolución estable. Sistema en Ejecución.")
	log.Fatal(http.ListenAndServe(":10000", nil))
}
