// pkg/uid/uid.go
// Générateur UUID v4 natif — zéro dépendance externe.
// Utilise crypto/rand pour une entropie cryptographiquement sûre.
package uid

import (
	"crypto/rand"
	"fmt"
)

// New génère un UUID v4 sous forme de string (xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx).
func New() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand ne devrait jamais échouer sur un OS sain.
		panic(fmt.Sprintf("uid: rand.Read failed: %v", err))
	}
	// Positionner les bits version (4) et variante (RFC 4122).
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}