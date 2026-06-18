package main

import (
    "fmt"
    "net/http"
    "log"
)

// Middleware para permitir que el Vue hable con el Córtex
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

func evolucionarCortex(w http.ResponseWriter, r *http.Request) {
    // Aquí recibiremos el JSON de la "Cajita"
    fmt.Fprintf(w, "Evolución aplicada exitosamente")
}

func main() {
    // Aplicamos el middleware a la ruta
    http.HandleFunc("/api/cortex/evolucionar", enableCors(evolucionarCortex))
    
    fmt.Println("🚀 [CÓRTEX]: Evolución estable. Sistema en Ejecución.")
    log.Fatal(http.ListenAndServe(":10000", nil))
}