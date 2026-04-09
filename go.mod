module github.com/axiom-ide/axiom

go 1.22
EOF

# Create an internal uuid package
mkdir -p /home/claude/axiom/internal/uid
cat > /home/claude/axiom/internal/uid/uid.go << 'GOEOF'
// Package uid fournit un générateur d'UUIDs v4 sans dépendances externes.
package uid

import (
	"crypto/rand"
	"fmt"
)

// New génère un UUID v4 aléatoire.
func New() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic("uid: cannot generate random bytes: " + err.Error())
	}
	// Version 4 (random)
	b[6] = (b[6] & 0x0f) | 0x40
	// Variant bits
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}